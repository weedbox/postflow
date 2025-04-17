# Postflow

[![License](https://img.shields.io/badge/license-APL-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/weedbox/postflow)](https://goreportcard.com/report/github.com/weedbox/postflow)
[![GoDoc](https://godoc.org/github.com/weedbox/postflow?status.svg)](https://godoc.org/github.com/weedbox/postflow)

Postflow is a flexible and extensible Go package for managing user posts and social media feeds. It provides a complete solution for creating, retrieving, updating, and deleting posts, as well as handling user interactions like reactions, comments, and shares.

## Features

- 📝 Complete post management (CRUD operations)
- 🔄 Feed generation and retrieval
- 🔍 Advanced post filtering and sorting
- 👍 Reaction management (like, love, haha, wow, sad, angry)
- 🏷️ Tag-based post organization
- 🖼️ Media attachment support (images, videos, audio, files, links)
- 🔒 Visibility control (public, private, friends)
- 📊 Trend detection and trending post retrieval
- 💾 Multiple storage options (in-memory and GORM-based database backends)

## Installation

```bash
go get github.com/weedbox/postflow
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/weedbox/postflow"
	"time"
)

func main() {
	// Create an in-memory store
	store := postflow.NewInMemoryPostStore()
	
	// Create a post manager
	manager := postflow.NewPostManager(store)
	
	// Create a new post
	post := &postflow.Post{
		UserID:     "user123",
		Content:    "Hello, Postflow!",
		Tags:       []string{"hello", "example"},
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	ctx := context.Background()
	
	// Save the post
	postID, err := manager.CreatePost(ctx, post)
	if err != nil {
		fmt.Printf("Error creating post: %v\n", err)
		return
	}
	
	fmt.Printf("Created post with ID: %s\n", postID)
	
	// Add a reaction to the post
	err = manager.AddReaction(ctx, postID, "user456", postflow.ReactionLike)
	if err != nil {
		fmt.Printf("Error adding reaction: %v\n", err)
		return
	}
	
	// Get the post with reaction
	retrievedPost, err := manager.GetPost(ctx, postID)
	if err != nil {
		fmt.Printf("Error retrieving post: %v\n", err)
		return
	}
	
	fmt.Printf("Post content: %s\n", retrievedPost.Content)
	fmt.Printf("Likes: %d\n", retrievedPost.Reactions[postflow.ReactionLike])
}
```

## Storage Options

### In-Memory Storage

For development, testing, or simple applications, you can use the `InMemoryPostStore`:

```go
store := postflow.NewInMemoryPostStore()
manager := postflow.NewPostManager(store)
```

### Database Storage with GORM

For production applications, you can use the GORM-based store, which supports various database backends:

```go
import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect to database
db, err := gorm.Open(postgres.Open("host=localhost user=postgres password=postgres dbname=postflow port=5432 sslmode=disable"), &gorm.Config{})
if err != nil {
	panic("failed to connect to database")
}

// Create a GORM store
store, err := postflow.NewGormPostStore(db)
if err != nil {
	panic("failed to create store")
}

manager := postflow.NewPostManager(store)
```

## Post Filtering

The package provides a flexible filtering system for retrieving posts:

```go
// Get posts by a specific user
posts, err := manager.ListPosts(ctx, &postflow.PostFilter{
	UserID: "user123",
})

// Get posts with specific tags
posts, err := manager.ListPosts(ctx, &postflow.PostFilter{
	Tags: []string{"golang", "programming"},
})

// Get posts within a time range
posts, err := manager.ListPosts(ctx, &postflow.PostFilter{
	TimeRange: &postflow.TimeRange{
		Start: time.Now().Add(-24 * time.Hour), // Last 24 hours
		End:   time.Now(),
	},
})

// Get posts with pagination and sorting
posts, err := manager.ListPosts(ctx, &postflow.PostFilter{
	Limit:     10,
	Offset:    0,
	SortBy:    "created_at",
	SortOrder: "desc",
})
```

## Reaction Types

The package supports several reaction types:

```go
postflow.ReactionLike  // 👍
postflow.ReactionLove  // ❤️
postflow.ReactionHaha  // 😄
postflow.ReactionWow   // 😮
postflow.ReactionSad   // 😢
postflow.ReactionAngry // 😠
```

## Media Support

Posts can include various types of media:

```go
post := &postflow.Post{
	UserID:  "user123",
	Content: "Check out this photo!",
	Media: []postflow.Media{
		{
			ID:           uuid.New().String(),
			Type:         postflow.MediaTypeImage,
			URL:          "https://example.com/image.jpg",
			ThumbnailURL: "https://example.com/thumbnail.jpg",
			Description:  "Beautiful sunset",
			Width:        1920,
			Height:       1080,
			FileSize:     1024 * 1024, // 1MB
			FileName:     "sunset.jpg",
			MimeType:     "image/jpeg",
			CreatedAt:    time.Now(),
		},
	},
	Visibility: "public",
}
```

## Feed Generation

Get a user's feed or trending posts:

```go
// Get a user's feed
feed, err := manager.GetUserFeed(ctx, "user123", 20, 0)

// Get trending posts
trending, err := manager.GetTrendingPosts(ctx, 10)
```

## Error Handling

The package defines several error types that you should handle in your application:

```go
if err == postflow.ErrPostNotFound {
	// Handle post not found
}

if err == postflow.ErrPermissionDenied {
	// Handle permission denied
}

if err == postflow.ErrInvalidReaction {
	// Handle invalid reaction
}
```

## Testing

To run the tests:

```bash
go test -v ./...
```

## License

This project is licensed under the APL License - see the [LICENSE](LICENSE) file for details.
