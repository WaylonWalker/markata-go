---
title: "Searchcraft on Kubernetes"
description: "Deploy Searchcraft securely on Kubernetes for markata-go ingestion and public search."
date: 2026-03-02
published: true
tags:
  - documentation
  - search
  - searchcraft
  - kubernetes
---

# Searchcraft on Kubernetes

This guide shows a secure Kubernetes setup where:

- markata-go build jobs can ingest/update/delete documents
- public users can run search queries
- public traffic cannot write to your index

## Security model

- Use two keys:
  - `SEARCHCRAFT_INGEST_KEY` (private, build/CI only)
  - `SEARCHCRAFT_READ_KEY` (public, search-only)
- Expose only search endpoints publicly.
- Keep write endpoints on an internal ingress or private service path.

## 1) Core resources

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: searchcraft-data
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: searchcraft
spec:
  replicas: 1
  selector:
    matchLabels:
      app: searchcraft
  template:
    metadata:
      labels:
        app: searchcraft
    spec:
      containers:
        - name: searchcraft
          image: searchcraftinc/searchcraft-core:latest
          args: ["--port", "18000"]
          ports:
            - containerPort: 18000
          volumeMounts:
            - name: data
              mountPath: /data
          readinessProbe:
            httpGet:
              path: /healthcheck
              port: 18000
            initialDelaySeconds: 5
            periodSeconds: 10
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: searchcraft-data
---
apiVersion: v1
kind: Service
metadata:
  name: searchcraft
spec:
  selector:
    app: searchcraft
  ports:
    - name: http
      port: 18000
      targetPort: 18000
```

## 2) Public read ingress and private write ingress

Use two hostnames:

- `search.example.com` (public read only)
- `search-write.internal.example.com` (private for CI/build)

Example (NGINX ingress):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: searchcraft-public
  annotations:
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      if ($request_method != POST) { return 405; }
spec:
  ingressClassName: nginx
  rules:
    - host: search.example.com
      http:
        paths:
          - path: /index/.*/search
            pathType: ImplementationSpecific
            backend:
              service:
                name: searchcraft
                port:
                  number: 18000
          - path: /healthcheck
            pathType: Exact
            backend:
              service:
                name: searchcraft
                port:
                  number: 18000
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: searchcraft-write
spec:
  ingressClassName: internal-nginx
  rules:
    - host: search-write.internal.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: searchcraft
                port:
                  number: 18000
```

## 3) NetworkPolicy (only ingress controller + CI namespace)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: searchcraft-ingress-policy
spec:
  podSelector:
    matchLabels:
      app: searchcraft
  policyTypes: ["Ingress"]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 18000
    - from:
        - namespaceSelector:
            matchLabels:
              name: ci
      ports:
        - protocol: TCP
          port: 18000
```

## 4) Keys and build secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: markata-searchcraft
type: Opaque
stringData:
  SEARCHCRAFT_INGEST_KEY: "replace-me-ingest"
  SEARCHCRAFT_READ_KEY: "replace-me-read"
```

Mount the secret into your build job that runs `markata-go build`.

## 5) markata-go config

### Same cluster (build job in Kubernetes)

```toml
[markata-go.searchcraft]
enabled = true
endpoint = "http://searchcraft.default.svc.cluster.local:18000"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_per_site = true
```

### Separate host (external Searchcraft)

```toml
[markata-go.searchcraft]
enabled = true
endpoint = "https://search-write.internal.example.com"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_per_site = true
delete_missing = true
```

## 6) Validate

```bash
kubectl get pods -l app=searchcraft
kubectl get ingress searchcraft-public searchcraft-write
curl -sS https://search.example.com/healthcheck
```

Then run a site build and verify index updates:

```bash
markata-go build -c markata-go.toml
```

## Notes

- Keep `SEARCHCRAFT_INGEST_KEY` out of frontend templates.
- Rotate both keys periodically.
- Back up the Searchcraft PVC.
