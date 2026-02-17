# Garden View Specification

The garden view plugin provides a digital garden / knowledge graph experience for discovering content by relationships rather than chronology. It exports a relationship graph as JSON and optionally renders an interactive garden page.

## Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                          GARDEN VIEW                                  │
│                                                                       │
│  Phase 1: Export graph.json                                           │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │  nodes: posts + tags                                          │   │
│  │  edges: post→post (links), post→tag, tag↔tag (co-occurrence) │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Phase 2: Render garden/index.html                                   │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │  Interactive page from garden.html template                    │   │
│  │  Tag clusters with related tags                                │   │
│  │  Related posts per post                                        │   │
│  └───────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

## Configuration

```toml
[markata-go.garden]
enabled = true              # Enable the garden view plugin (default: true)
path = "garden"             # Output path prefix (default: "garden")
export_json = true          # Emit graph.json (default: true)
render_page = true          # Generate garden/index.html (default: true)
include_tags = true         # Include tag nodes in graph (default: true)
include_posts = true        # Include post nodes in graph (default: true)
max_nodes = 2000            # Maximum number of nodes (default: 2000)
exclude_tags = []           # Tags to exclude from graph (default: [])
template = "garden.html"    # Template for the garden page (default: "garden.html")
title = "Garden"            # Page title (default: "Garden")
description = ""            # Page description (default: "")
```

## Data Model

### Graph JSON Schema

The `graph.json` output file has the following structure:

```json
{
  "nodes": [
    {
      "id": "post:my-post-slug",
      "type": "post",
      "label": "My Post Title",
      "href": "/my-post-slug/",
      "tags": ["go", "programming"],
      "date": "2024-01-15T00:00:00Z",
      "description": "A short description"
    },
    {
      "id": "tag:go",
      "type": "tag",
      "label": "go",
      "href": "/tags/go/",
      "count": 15
    }
  ],
  "edges": [
    {
      "source": "post:my-post-slug",
      "target": "post:another-post",
      "type": "link"
    },
    {
      "source": "post:my-post-slug",
      "target": "tag:go",
      "type": "tag"
    },
    {
      "source": "tag:go",
      "target": "tag:programming",
      "type": "co-occurrence",
      "weight": 8
    }
  ]
}
```

### Node Types

| Type   | ID Format           | Fields                                  |
|--------|---------------------|-----------------------------------------|
| `post` | `post:<slug>`       | label, href, tags, date, description    |
| `tag`  | `tag:<tag-name>`    | label, href, count                      |

### Edge Types

| Type            | Source → Target  | Description                              |
|-----------------|------------------|------------------------------------------|
| `link`          | post → post      | Internal link from one post to another   |
| `tag`           | post → tag       | Post belongs to tag                      |
| `co-occurrence` | tag → tag        | Tags appear together on posts (weighted) |

## Plugin Behavior

### Stage: Write (PriorityLate)

The garden view plugin runs during the Write stage at `PriorityLate` priority, after the link collector and feeds have finished processing.

### Post Filtering

Posts are included in the graph if they meet ALL criteria:
- `published == true`
- `draft == false`
- `private == false`
- `skip == false`
- None of their tags are in the `exclude_tags` list

### Tag Co-occurrence

Two tags co-occur when they appear on the same post. The weight of a co-occurrence edge is the number of posts where both tags appear together. This enables "related tags" discovery.

### Node Limit

When the total number of nodes exceeds `max_nodes`, tags are sorted by post count and the least-used tags are removed first. Posts are never removed by the node limit.

### Deterministic Output

The `graph.json` file must be deterministic (same input produces same output). Nodes are sorted by ID, edges are sorted by (source, target, type).

## Template

The garden page uses the `garden.html` template with the following context variables:

| Variable      | Type       | Description                        |
|---------------|------------|------------------------------------|
| `title`       | string     | Page title                         |
| `description` | string     | Page description                   |
| `graph_json`  | string     | Path to graph.json (relative URL)  |
| `tag_clusters`| []TagCluster | Tag groups with related tags     |
| `total_posts` | int        | Number of posts in the graph       |
| `total_tags`  | int        | Number of tags in the graph        |
| `total_edges` | int        | Number of edges in the graph       |

### TagCluster

| Field        | Type     | Description                       |
|--------------|----------|-----------------------------------|
| `name`       | string   | Tag name                          |
| `count`      | int      | Number of posts with this tag     |
| `href`       | string   | URL to the tag page               |
| `related`    | []string | Names of co-occurring tags        |

## Dependencies

- **link_collector** plugin (Render stage) - provides post-to-post link data
- **tag_aggregator** plugin (Load stage) - provides normalized tags
- **tags_listing** plugin (Write stage) - shares tag URL patterns

## File Output

| Config           | Output Path                    |
|------------------|--------------------------------|
| `export_json`    | `/<path>/graph.json`           |
| `render_page`    | `/<path>/index.html`           |
