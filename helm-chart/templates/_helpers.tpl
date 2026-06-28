{{- define "markata-notes.namespace" -}}
{{- printf "%s-notes" .Values.project_identifier -}}
{{- end -}}

{{- define "markata-notes.projectName" -}}
{{- default .Values.project_identifier .Values.project_name -}}
{{- end -}}

{{- define "markata-notes.labels" -}}
app.kubernetes.io/name: "markata-notes"
app.kubernetes.io/instance: "{{ .Values.project_identifier }}"
app.kubernetes.io/component: "notes"
app.kubernetes.io/part-of: "{{ include "markata-notes.projectName" . }}"
app.kubernetes.io/environment: "{{ .Values.environment | default "prod" }}"
app.kubernetes.io/managed-by: "{{ .Release.Service }}"
helm.sh/chart: "{{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}"
{{- end -}}

{{- define "markata-notes.sitePvcName" -}}
{{- printf "%s-notes-site%s-pvc" .Values.project_identifier (.Values.storage.site.pvcNameSuffix | default "") -}}
{{- end -}}

{{- define "markata-notes.sourcePvcName" -}}
{{- printf "%s-notes-source%s-pvc" .Values.project_identifier (.Values.storage.source.pvcNameSuffix | default "") -}}
{{- end -}}

{{- define "markata-notes.searchPvcName" -}}
{{- printf "%s-notes-search%s-pvc" .Values.project_identifier (.Values.storage.search.pvcNameSuffix | default "") -}}
{{- end -}}

{{- define "markata-notes.cachePvcName" -}}
{{- printf "%s-notes-cache%s-pvc" .Values.project_identifier (.Values.storage.cache.pvcNameSuffix | default "") -}}
{{- end -}}

{{- define "markata-notes.host" -}}
{{- default (printf "%s.example.com" .Values.project_identifier) .Values.ingress.host -}}
{{- end -}}

{{- define "markata-notes.tlsSecretName" -}}
{{- default (printf "%s-notes-tls" .Values.project_identifier) .Values.ingress.tls.secretName -}}
{{- end -}}

{{- define "markata-notes.serviceAccountName" -}}
{{- if .Values.serviceAccount.name -}}
{{- .Values.serviceAccount.name -}}
{{- else -}}
{{- printf "%s-notes-workload" .Values.project_identifier -}}
{{- end -}}
{{- end -}}
