// Package postflow provides functionality for managing user posts and feeds.
package postflow

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	// ErrPostNotFound is returned when a post is not found
	ErrPostNotFound = errors.New("post not found")

	// ErrPermissionDenied is returned when a user doesn't have permission to modify a post
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidReaction is returned when an invalid reaction is provided
	ErrInvalidReaction = errors.New("invalid reaction")
)

// PostStore defines the interface for storing and retrieving posts
type PostStore interface {
	// SavePost saves a new post or updates an existing post
	SavePost(ctx context.Context, post *Post) error

	// GetPost retrieves a post by its ID
	GetPost(ctx context.Context, postID string) (*Post, error)

	// DeletePost removes a post from the store
	DeletePost(ctx context.Context, postID string, userID string) error

	// ListPosts retrieves posts based on filter criteria
	ListPosts(ctx context.Context, filter *PostFilter) ([]*Post, error)

	// GetUserFeed retrieves posts for a user's feed
	GetUserFeed(ctx context.Context, userID string, limit, offset int) ([]*Post, error)

	// GetTrendingPosts retrieves currently trending posts
	GetTrendingPosts(ctx context.Context, limit int) ([]*Post, error)

	// SaveReaction saves a reaction to a post
	SaveReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error

	// DeleteReaction removes a reaction from a post
	DeleteReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error

	// GetUserReaction gets the current reaction of a user for a post
	GetUserReaction(ctx context.Context, postID string, userID string) (*ReactionType, error)

	// GetReactedUsers returns users who reacted to a specific post with optional reaction type filter
	GetReactedUsers(ctx context.Context, postID string, reactionType *ReactionType, limit, offset int) ([]string, error)

	// GetReactionCounts returns the count of each reaction type for a post
	GetReactionCounts(ctx context.Context, postID string) (map[ReactionType]int, error)
}

// UserReaction represents a user's reaction to a post
type UserReaction struct {
	UserID       string
	ReactionType ReactionType
	CreatedAt    time.Time
}

// InMemoryPostStore implements PostStore interface with in-memory storage
type InMemoryPostStore struct {
	mutex     sync.RWMutex
	posts     map[string]*Post                    // postID -> Post
	reactions map[string]map[string]*UserReaction // postID -> userID -> UserReaction
	userPosts map[string][]string                 // userID -> []postID
	tagPosts  map[string][]string                 // tag -> []postID
}

// NewInMemoryPostStore creates a new instance of InMemoryPostStore
func NewInMemoryPostStore() *InMemoryPostStore {
	return &InMemoryPostStore{
		posts:     make(map[string]*Post),
		reactions: make(map[string]map[string]*UserReaction),
		userPosts: make(map[string][]string),
		tagPosts:  make(map[string][]string),
	}
}

// SavePost saves a post to the in-memory store
func (s *InMemoryPostStore) SavePost(ctx context.Context, post *Post) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.posts[post.ID]; !exists {
		// New post
		s.posts[post.ID] = post

		// Index by user
		s.userPosts[post.UserID] = append(s.userPosts[post.UserID], post.ID)

		// Index by tags
		for _, tag := range post.Tags {
			s.tagPosts[tag] = append(s.tagPosts[tag], post.ID)
		}
	} else {
		// Update existing post
		oldPost := s.posts[post.ID]

		// Remove old tag references
		for _, tag := range oldPost.Tags {
			tagPosts := s.tagPosts[tag]
			for i, pid := range tagPosts {
				if pid == post.ID {
					s.tagPosts[tag] = append(tagPosts[:i], tagPosts[i+1:]...)
					break
				}
			}
		}

		// Add new tag references
		for _, tag := range post.Tags {
			found := false
			for _, pid := range s.tagPosts[tag] {
				if pid == post.ID {
					found = true
					break
				}
			}
			if !found {
				s.tagPosts[tag] = append(s.tagPosts[tag], post.ID)
			}
		}

		// Update the post
		s.posts[post.ID] = post
	}

	return nil
}

// GetPost retrieves a post by its ID
func (s *InMemoryPostStore) GetPost(ctx context.Context, postID string) (*Post, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	post, exists := s.posts[postID]
	if !exists {
		return nil, ErrPostNotFound
	}

	// Return a copy to prevent modifications to the stored post
	postCopy := *post
	return &postCopy, nil
}

// DeletePost removes a post from the store
func (s *InMemoryPostStore) DeletePost(ctx context.Context, postID string, userID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	post, exists := s.posts[postID]
	if !exists {
		return ErrPostNotFound
	}

	// Check if the user is authorized to delete the post
	if post.UserID != userID {
		return ErrPermissionDenied
	}

	// Remove from user posts
	userPosts := s.userPosts[userID]
	for i, pid := range userPosts {
		if pid == postID {
			s.userPosts[userID] = append(userPosts[:i], userPosts[i+1:]...)
			break
		}
	}

	// Remove from tag posts
	for _, tag := range post.Tags {
		tagPosts := s.tagPosts[tag]
		for i, pid := range tagPosts {
			if pid == postID {
				s.tagPosts[tag] = append(tagPosts[:i], tagPosts[i+1:]...)
				break
			}
		}
	}

	// Remove reactions
	delete(s.reactions, postID)

	// Remove the post
	delete(s.posts, postID)

	return nil
}

// ListPosts retrieves posts based on filter criteria
func (s *InMemoryPostStore) ListPosts(ctx context.Context, filter *PostFilter) ([]*Post, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*Post
	var candidateIDs map[string]bool

	// Start with user filter if present
	if filter.UserID != "" {
		candidateIDs = make(map[string]bool)
		for _, pid := range s.userPosts[filter.UserID] {
			candidateIDs[pid] = true
		}
	}

	// Filter by tags if present
	if len(filter.Tags) > 0 {
		tagCandidates := make(map[string]bool)

		for i, tag := range filter.Tags {
			for _, pid := range s.tagPosts[tag] {
				if i == 0 || candidateIDs[pid] {
					tagCandidates[pid] = true
				}
			}
		}

		candidateIDs = tagCandidates
	}

	// If no specific filters were applied, consider all posts
	if candidateIDs == nil {
		for id := range s.posts {
			post := s.posts[id]

			// Apply visibility filter
			if filter.Visibility != "" && post.Visibility != filter.Visibility {
				continue
			}

			// Apply time range filter
			if filter.TimeRange != nil {
				if !post.CreatedAt.After(filter.TimeRange.Start) || !post.CreatedAt.Before(filter.TimeRange.End) {
					continue
				}
			}

			postCopy := *post
			result = append(result, &postCopy)
		}
	} else {
		// Apply additional filters to candidate posts
		for id := range candidateIDs {
			post, exists := s.posts[id]
			if !exists {
				continue
			}

			// Apply visibility filter
			if filter.Visibility != "" && post.Visibility != filter.Visibility {
				continue
			}

			// Apply time range filter
			if filter.TimeRange != nil {
				if !post.CreatedAt.After(filter.TimeRange.Start) || !post.CreatedAt.Before(filter.TimeRange.End) {
					continue
				}
			}

			postCopy := *post
			result = append(result, &postCopy)
		}
	}

	// Sort results
	if filter.SortBy != "" {
		sort.Slice(result, func(i, j int) bool {
			var less bool

			switch filter.SortBy {
			case "created_at":
				less = result[i].CreatedAt.Before(result[j].CreatedAt)
			case "updated_at":
				less = result[i].UpdatedAt.Before(result[j].UpdatedAt)
			case "reactions":
				less = sumReactions(result[i].Reactions) < sumReactions(result[j].Reactions)
			case "comments":
				less = result[i].Comments < result[j].Comments
			case "shares":
				less = result[i].Shares < result[j].Shares
			default:
				// Default sort by created_at
				less = result[i].CreatedAt.Before(result[j].CreatedAt)
			}

			// Invert for descending order
			if filter.SortOrder == "desc" {
				return !less
			}
			return less
		})
	}

	// Apply pagination
	if filter.Limit > 0 {
		end := filter.Offset + filter.Limit
		if end > len(result) {
			end = len(result)
		}
		if filter.Offset < len(result) {
			result = result[filter.Offset:end]
		} else {
			result = []*Post{}
		}
	}

	return result, nil
}

// GetUserFeed retrieves posts for a user's feed
// In a real implementation, this would consider followed users, algorithms, etc.
// This simple version just returns recent public posts
func (s *InMemoryPostStore) GetUserFeed(ctx context.Context, userID string, limit, offset int) ([]*Post, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*Post

	// Get all public posts
	for _, post := range s.posts {
		if post.Visibility == "public" {
			postCopy := *post
			result = append(result, &postCopy)
		}
	}

	// Sort by creation time, newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply pagination
	if limit > 0 {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		if offset < len(result) {
			result = result[offset:end]
		} else {
			result = []*Post{}
		}
	}

	return result, nil
}

// GetTrendingPosts retrieves currently trending posts
// This simple implementation returns posts with most reactions and comments
func (s *InMemoryPostStore) GetTrendingPosts(ctx context.Context, limit int) ([]*Post, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*Post

	// Get all public posts
	for _, post := range s.posts {
		if post.Visibility == "public" {
			postCopy := *post
			result = append(result, &postCopy)
		}
	}

	// Sort by engagement (reactions + comments + shares)
	sort.Slice(result, func(i, j int) bool {
		engagementI := sumReactions(result[i].Reactions) + result[i].Comments + result[i].Shares
		engagementJ := sumReactions(result[j].Reactions) + result[j].Comments + result[j].Shares
		return engagementI > engagementJ
	})

	// Apply limit
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// SaveReaction saves a reaction to a post
func (s *InMemoryPostStore) SaveReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	post, exists := s.posts[postID]
	if !exists {
		return ErrPostNotFound
	}

	// Check if reaction type is valid
	if reactionType <= ReactionNone || reactionType > ReactionAngry {
		return ErrInvalidReaction
	}

	// Initialize reactions map for this post if it doesn't exist
	if _, exists := s.reactions[postID]; !exists {
		s.reactions[postID] = make(map[string]*UserReaction)
	}

	// Check if user already has a reaction
	existingReaction, hasReaction := s.reactions[postID][userID]

	if hasReaction {
		// Update reaction counts in post
		if existingReaction.ReactionType != reactionType {
			// Decrement old reaction count
			if post.Reactions == nil {
				post.Reactions = make(map[ReactionType]int)
			}
			oldCount := post.Reactions[existingReaction.ReactionType]
			if oldCount > 0 {
				post.Reactions[existingReaction.ReactionType] = oldCount - 1
			}

			// Increment new reaction count
			post.Reactions[reactionType] = post.Reactions[reactionType] + 1

			// Update reaction
			existingReaction.ReactionType = reactionType
			existingReaction.CreatedAt = time.Now()
		}
	} else {
		// Add new reaction
		s.reactions[postID][userID] = &UserReaction{
			UserID:       userID,
			ReactionType: reactionType,
			CreatedAt:    time.Now(),
		}

		// Update reaction count in post
		if post.Reactions == nil {
			post.Reactions = make(map[ReactionType]int)
		}
		post.Reactions[reactionType] = post.Reactions[reactionType] + 1
	}

	return nil
}

// DeleteReaction removes a reaction from a post
func (s *InMemoryPostStore) DeleteReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	post, exists := s.posts[postID]
	if !exists {
		return ErrPostNotFound
	}

	// Check if reaction exists
	postReactions, postExists := s.reactions[postID]
	if !postExists {
		return nil // No reactions for this post
	}

	userReaction, userExists := postReactions[userID]
	if !userExists || userReaction.ReactionType != reactionType {
		return nil // User doesn't have this reaction
	}

	// Remove reaction
	delete(postReactions, userID)

	// Update reaction count in post
	if post.Reactions != nil && post.Reactions[reactionType] > 0 {
		post.Reactions[reactionType] = post.Reactions[reactionType] - 1
	}

	return nil
}

// GetUserReaction gets the current reaction of a user for a post
func (s *InMemoryPostStore) GetUserReaction(ctx context.Context, postID string, userID string) (*ReactionType, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if _, exists := s.posts[postID]; !exists {
		return nil, ErrPostNotFound
	}

	postReactions, exists := s.reactions[postID]
	if !exists {
		return nil, nil // No reactions for this post
	}

	userReaction, exists := postReactions[userID]
	if !exists {
		return nil, nil // User hasn't reacted
	}

	reaction := userReaction.ReactionType
	return &reaction, nil
}

// GetReactedUsers returns users who reacted to a specific post
func (s *InMemoryPostStore) GetReactedUsers(ctx context.Context, postID string, reactionType *ReactionType, limit, offset int) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if _, exists := s.posts[postID]; !exists {
		return nil, ErrPostNotFound
	}

	postReactions, exists := s.reactions[postID]
	if !exists {
		return []string{}, nil // No reactions for this post
	}

	var userIDs []string

	// Filter by reaction type if specified
	for userID, reaction := range postReactions {
		if reactionType == nil || reaction.ReactionType == *reactionType {
			userIDs = append(userIDs, userID)
		}
	}

	// Sort by reaction time (most recent first)
	sort.Slice(userIDs, func(i, j int) bool {
		return postReactions[userIDs[i]].CreatedAt.After(postReactions[userIDs[j]].CreatedAt)
	})

	// Apply pagination
	if limit > 0 {
		end := offset + limit
		if end > len(userIDs) {
			end = len(userIDs)
		}
		if offset < len(userIDs) {
			userIDs = userIDs[offset:end]
		} else {
			userIDs = []string{}
		}
	}

	return userIDs, nil
}

// GetReactionCounts returns the count of each reaction type for a post
func (s *InMemoryPostStore) GetReactionCounts(ctx context.Context, postID string) (map[ReactionType]int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	post, exists := s.posts[postID]
	if !exists {
		return nil, ErrPostNotFound
	}

	// Create a copy of the reactions map
	if post.Reactions == nil {
		return make(map[ReactionType]int), nil
	}

	result := make(map[ReactionType]int)
	for reactionType, count := range post.Reactions {
		result[reactionType] = count
	}

	return result, nil
}

// Helper function to sum all reactions
func sumReactions(reactions map[ReactionType]int) int {
	var sum int
	for _, count := range reactions {
		sum += count
	}
	return sum
}
