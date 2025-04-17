// Package postflow provides functionality for managing user posts and feeds.
//
// The package implements a post management system that supports creating, retrieving,
// updating and deleting posts, as well as managing user feeds and interactions.
package postflow

import (
	"context"
	"time"
)

// ReactionType represents different emotional reactions to posts using an efficient numeric type
type ReactionType uint8

const (
	ReactionNone  ReactionType = 0
	ReactionLike  ReactionType = 1
	ReactionLove  ReactionType = 2
	ReactionHaha  ReactionType = 3
	ReactionWow   ReactionType = 4
	ReactionSad   ReactionType = 5
	ReactionAngry ReactionType = 6
)

// MediaType represents the type of media attached to a post
type MediaType uint8

const (
	MediaTypeImage MediaType = 1
	MediaTypeVideo MediaType = 2
	MediaTypeAudio MediaType = 3
	MediaTypeFile  MediaType = 4
	MediaTypeLink  MediaType = 5
)

// Media represents a media item attached to a post
type Media struct {
	ID           string    `json:"id"`
	Type         MediaType `json:"type"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Description  string    `json:"description,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Duration     int       `json:"duration,omitempty"`  // Duration in seconds for video/audio
	FileSize     int64     `json:"file_size,omitempty"` // Size in bytes
	FileName     string    `json:"file_name,omitempty"`
	MimeType     string    `json:"mime_type,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Post represents a user post in the system.
type Post struct {
	ID         string               `json:"id"`
	UserID     string               `json:"user_id"`
	Content    string               `json:"content"`
	Media      []Media              `json:"media,omitempty"`
	Tags       []string             `json:"tags,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Reactions  map[ReactionType]int `json:"reactions"`
	Comments   int                  `json:"comments"`
	Shares     int                  `json:"shares"`
	Visibility string               `json:"visibility"` // public, private, friends
}

// PostFilter represents filtering options for retrieving posts.
type PostFilter struct {
	UserID     string
	Tags       []string
	TimeRange  *TimeRange
	Visibility string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// TimeRange represents a time range for filtering posts.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// PostManager defines the interface for managing posts in the system.
type PostManager interface {
	// CreatePost creates a new post in the system.
	CreatePost(ctx context.Context, post *Post) (string, error)

	// GetPost retrieves a post by its ID.
	GetPost(ctx context.Context, postID string) (*Post, error)

	// UpdatePost updates an existing post.
	UpdatePost(ctx context.Context, post *Post) error

	// DeletePost removes a post from the system.
	DeletePost(ctx context.Context, postID string, userID string) error

	// ListPosts retrieves a list of posts based on filter criteria.
	ListPosts(ctx context.Context, filter *PostFilter) ([]*Post, error)

	// GetUserFeed returns posts for a user's feed.
	GetUserFeed(ctx context.Context, userID string, limit, offset int) ([]*Post, error)

	// GetTrendingPosts returns currently trending posts.
	GetTrendingPosts(ctx context.Context, limit int) ([]*Post, error)

	// AddReaction adds an emotional reaction to a post.
	AddReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error

	// RemoveReaction removes an emotional reaction from a post.
	RemoveReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error

	// GetUserReaction gets the current reaction of a user for a post.
	GetUserReaction(ctx context.Context, postID string, userID string) (*ReactionType, error)

	// GetReactedUsers returns users who reacted to a specific post with optional reaction type filter.
	GetReactedUsers(ctx context.Context, postID string, reactionType *ReactionType, limit, offset int) ([]string, error)

	// GetReactionCounts returns the count of each reaction type for a post.
	GetReactionCounts(ctx context.Context, postID string) (map[ReactionType]int, error)
}
