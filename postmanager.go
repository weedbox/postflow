// Package postflow provides functionality for managing user posts and feeds.
package postflow

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PostManagerImpl implements the PostManager interface using a PostStore for persistence
type PostManagerImpl struct {
	store PostStore
}

// NewPostManager creates a new instance of PostManagerImpl
func NewPostManager(store PostStore) *PostManagerImpl {
	return &PostManagerImpl{
		store: store,
	}
}

// CreatePost creates a new post in the system
func (m *PostManagerImpl) CreatePost(ctx context.Context, post *Post) (string, error) {
	// Validate the post
	if post.UserID == "" {
		return "", errors.New("user ID is required")
	}

	// Generate a new ID if not provided
	if post.ID == "" {
		post.ID = uuid.New().String()
	}

	// Set creation time
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// Initialize reactions map if not present
	if post.Reactions == nil {
		post.Reactions = make(map[ReactionType]int)
	}

	// Save to store
	err := m.store.SavePost(ctx, post)
	if err != nil {
		return "", err
	}

	return post.ID, nil
}

// GetPost retrieves a post by its ID
func (m *PostManagerImpl) GetPost(ctx context.Context, postID string) (*Post, error) {
	return m.store.GetPost(ctx, postID)
}

// UpdatePost updates an existing post
func (m *PostManagerImpl) UpdatePost(ctx context.Context, post *Post) error {
	// Validate post
	if post.ID == "" {
		return errors.New("post ID is required")
	}

	// Check if post exists
	existingPost, err := m.store.GetPost(ctx, post.ID)
	if err != nil {
		return err
	}

	// Check if user is authorized to update the post
	if existingPost.UserID != post.UserID {
		return errors.New("unauthorized to update this post")
	}

	// Update modification time
	post.UpdatedAt = time.Now()

	// Preserve creation time
	post.CreatedAt = existingPost.CreatedAt

	// Save to store
	return m.store.SavePost(ctx, post)
}

// DeletePost removes a post from the system
func (m *PostManagerImpl) DeletePost(ctx context.Context, postID string, userID string) error {
	return m.store.DeletePost(ctx, postID, userID)
}

// ListPosts retrieves a list of posts based on filter criteria
func (m *PostManagerImpl) ListPosts(ctx context.Context, filter *PostFilter) ([]*Post, error) {
	return m.store.ListPosts(ctx, filter)
}

// GetUserFeed returns posts for a user's feed
func (m *PostManagerImpl) GetUserFeed(ctx context.Context, userID string, limit, offset int) ([]*Post, error) {
	return m.store.GetUserFeed(ctx, userID, limit, offset)
}

// GetTrendingPosts returns currently trending posts
func (m *PostManagerImpl) GetTrendingPosts(ctx context.Context, limit int) ([]*Post, error) {
	return m.store.GetTrendingPosts(ctx, limit)
}

// AddReaction adds an emotional reaction to a post
func (m *PostManagerImpl) AddReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	return m.store.SaveReaction(ctx, postID, userID, reactionType)
}

// RemoveReaction removes an emotional reaction from a post
func (m *PostManagerImpl) RemoveReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	return m.store.DeleteReaction(ctx, postID, userID, reactionType)
}

// GetUserReaction gets the current reaction of a user for a post
func (m *PostManagerImpl) GetUserReaction(ctx context.Context, postID string, userID string) (*ReactionType, error) {
	return m.store.GetUserReaction(ctx, postID, userID)
}

// GetReactedUsers returns users who reacted to a specific post with optional reaction type filter
func (m *PostManagerImpl) GetReactedUsers(ctx context.Context, postID string, reactionType *ReactionType, limit, offset int) ([]string, error) {
	return m.store.GetReactedUsers(ctx, postID, reactionType, limit, offset)
}

// GetReactionCounts returns the count of each reaction type for a post
func (m *PostManagerImpl) GetReactionCounts(ctx context.Context, postID string) (map[ReactionType]int, error) {
	return m.store.GetReactionCounts(ctx, postID)
}
