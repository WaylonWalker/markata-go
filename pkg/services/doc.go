// Package services provides reusable business logic interfaces for markata-go.
//
// # Overview
//
// This package defines service interfaces that abstract the lifecycle.Manager
// and provide clean APIs for TUI, CLI, and future web interfaces. All services
// are designed to be stateless and thread-safe.
//
// # Service Architecture
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│                    pkg/services/ (Business Logic)                │
//	│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
//	│  │ PostService │ │ FeedService │ │BuildService │               │
//	│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘               │
//	│         │               │               │                       │
//	│         └───────────────┴───────────────┘                       │
//	│                         │                                        │
//	│              ┌──────────┴──────────┐                            │
//	│              │  lifecycle.Manager   │                           │
//	│              └─────────────────────┘                            │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Usage
//
// Create an App instance to access all services:
//
//	app, err := services.NewApp(manager)
//	if err != nil {
//	    return err
//	}
//
//	posts, err := app.Posts.List(ctx, services.ListOptions{
//	    Published: boolPtr(true),
//	    SortBy:    "date",
//	    Limit:     10,
//	})
package services
