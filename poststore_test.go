package postflow

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// setupTestStore creates a new InMemoryPostStore with some test data
func setupTestStore() *InMemoryPostStore {
	store := NewInMemoryPostStore()
	return store
}

// createTestPost creates a test post with the given userID
func createTestPost(userID string) *Post {
	return &Post{
		ID:         uuid.New().String(),
		UserID:     userID,
		Content:    "Test content",
		Tags:       []string{"test", "golang"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Reactions:  make(map[ReactionType]int),
		Comments:   0,
		Shares:     0,
		Visibility: "public",
	}
}

// createTestPosts creates multiple test posts
func createTestPosts(store *InMemoryPostStore, userID string, count int) []*Post {
	ctx := context.Background()
	posts := make([]*Post, count)

	for i := 0; i < count; i++ {
		post := createTestPost(userID)
		post.Content = post.Content + " " + uuid.New().String()
		store.SavePost(ctx, post)
		posts[i] = post
	}

	return posts
}

// TestSavePost tests the SavePost method
func TestSavePost(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Test creating a new post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test retrieving the post
	savedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, post.ID, savedPost.ID)
	assert.Equal(t, post.UserID, savedPost.UserID)
	assert.Equal(t, post.Content, savedPost.Content)

	// Test updating a post
	post.Content = "Updated content"
	post.Tags = []string{"test", "golang", "updated"}
	err = store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test retrieving the updated post
	updatedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated content", updatedPost.Content)
	assert.Equal(t, 3, len(updatedPost.Tags))
	assert.Contains(t, updatedPost.Tags, "updated")
}

// TestGetPost tests the GetPost method
func TestGetPost(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test getting a post that exists
	foundPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, post.ID, foundPost.ID)

	// Test that the post returned is a copy
	foundPost.Content = "Modified in test"
	originalPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.NotEqual(t, foundPost.Content, originalPost.Content)

	// Test getting a post that doesn't exist
	_, err = store.GetPost(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestDeletePost tests the DeletePost method
func TestDeletePost(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test deleting by a different user (should fail)
	err = store.DeletePost(ctx, post.ID, "user2")
	assert.Error(t, err)
	assert.Equal(t, ErrPermissionDenied, err)

	// Verify post still exists
	_, err = store.GetPost(ctx, post.ID)
	assert.NoError(t, err)

	// Test deleting by correct user
	err = store.DeletePost(ctx, post.ID, "user1")
	assert.NoError(t, err)

	// Verify post is deleted
	_, err = store.GetPost(ctx, post.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)

	// Test deleting a nonexistent post
	err = store.DeletePost(ctx, "nonexistent-id", "user1")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestListPosts tests the ListPosts method
func TestListPosts(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create test posts for two users
	_ = createTestPosts(store, "user1", 5)
	_ = createTestPosts(store, "user2", 3)

	// Test: list posts by user1
	filter := &PostFilter{
		UserID: "user1",
	}
	posts, err := store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(posts))
	for _, post := range posts {
		assert.Equal(t, "user1", post.UserID)
	}

	// Test: list posts by user2
	filter.UserID = "user2"
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	// Test: list posts with tag filter
	filter = &PostFilter{
		Tags: []string{"test"},
	}
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(posts))

	// Test: list posts with pagination
	filter = &PostFilter{
		Limit:  3,
		Offset: 0,
	}
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	// Test: list posts with pagination (second page)
	filter.Offset = 3
	secondPage, err := store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(secondPage))
	assert.NotEqual(t, posts[0].ID, secondPage[0].ID)

	// Test: list posts with sorting (created_at, asc)
	filter = &PostFilter{
		SortBy:    "created_at",
		SortOrder: "asc",
	}
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.True(t, posts[0].CreatedAt.Before(posts[len(posts)-1].CreatedAt) ||
		posts[0].CreatedAt.Equal(posts[len(posts)-1].CreatedAt))

	// Test: list posts with sorting (created_at, desc)
	filter.SortOrder = "desc"
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.True(t, posts[0].CreatedAt.After(posts[len(posts)-1].CreatedAt) ||
		posts[0].CreatedAt.Equal(posts[len(posts)-1].CreatedAt))

	// Test: list posts with visibility filter
	// First, create a private post
	privatePost := createTestPost("user1")
	privatePost.Visibility = "private"
	store.SavePost(ctx, privatePost)

	filter = &PostFilter{
		Visibility: "private",
	}
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(posts))
	assert.Equal(t, "private", posts[0].Visibility)

	// Test: list posts with time range filter
	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	filter = &PostFilter{
		TimeRange: &TimeRange{
			Start: pastTime,
			End:   futureTime,
		},
	}
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Greater(t, len(posts), 0)
}

// TestGetUserFeed tests the GetUserFeed method
func TestGetUserFeed(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create test posts, some public, some private
	for i := 0; i < 5; i++ {
		post := createTestPost("user1")
		post.Visibility = "public"
		store.SavePost(ctx, post)
	}
	for i := 0; i < 3; i++ {
		post := createTestPost("user2")
		post.Visibility = "private"
		store.SavePost(ctx, post)
	}

	// Test: get user feed for user3 (should only see public posts)
	posts, err := store.GetUserFeed(ctx, "user3", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(posts))
	for _, post := range posts {
		assert.Equal(t, "public", post.Visibility)
	}

	// Test: pagination
	postsPage1, err := store.GetUserFeed(ctx, "user3", 2, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(postsPage1))

	postsPage2, err := store.GetUserFeed(ctx, "user3", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(postsPage2))

	// Ensure pages are different
	assert.NotEqual(t, postsPage1[0].ID, postsPage2[0].ID)
}

// TestGetTrendingPosts tests the GetTrendingPosts method
func TestGetTrendingPosts(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create test posts with different engagement levels
	post1 := createTestPost("user1")
	post1.Reactions = map[ReactionType]int{ReactionLike: 10, ReactionLove: 5}
	post1.Comments = 7
	post1.Shares = 3
	store.SavePost(ctx, post1)

	post2 := createTestPost("user1")
	post2.Reactions = map[ReactionType]int{ReactionLike: 5, ReactionHaha: 2}
	post2.Comments = 3
	post2.Shares = 1
	store.SavePost(ctx, post2)

	post3 := createTestPost("user2")
	post3.Reactions = map[ReactionType]int{ReactionLike: 20, ReactionWow: 8}
	post3.Comments = 12
	post3.Shares = 5
	store.SavePost(ctx, post3)

	// Test: get trending posts
	posts, err := store.GetTrendingPosts(ctx, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts))

	// The most engaging post should be first
	assert.Equal(t, post3.ID, posts[0].ID)

	// Test: limit
	allPosts, err := store.GetTrendingPosts(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(allPosts))
}

// TestSaveReaction tests the SaveReaction method
func TestSaveReaction(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test: save a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction count was updated
	updatedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedPost.Reactions[ReactionLike])

	// Test: change a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLove)
	assert.NoError(t, err)

	// Verify reaction counts were updated
	updatedPost, err = store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, updatedPost.Reactions[ReactionLike])
	assert.Equal(t, 1, updatedPost.Reactions[ReactionLove])

	// Test: invalid reaction type
	err = store.SaveReaction(ctx, post.ID, "user3", ReactionType(100))
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidReaction, err)

	// Test: reaction to nonexistent post
	err = store.SaveReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestDeleteReaction tests the DeleteReaction method
func TestDeleteReaction(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Add a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: delete the reaction
	err = store.DeleteReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction was removed
	updatedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, updatedPost.Reactions[ReactionLike])

	// Test: delete a nonexistent reaction (should not error)
	err = store.DeleteReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: delete reaction for nonexistent post
	err = store.DeleteReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestGetUserReaction tests the GetUserReaction method
func TestGetUserReaction(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Add a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: get user reaction that exists
	reaction, err := store.GetUserReaction(ctx, post.ID, "user2")
	assert.NoError(t, err)
	assert.NotNil(t, reaction)
	assert.Equal(t, ReactionLike, *reaction)

	// Test: get user reaction that doesn't exist
	reaction, err = store.GetUserReaction(ctx, post.ID, "user3")
	assert.NoError(t, err)
	assert.Nil(t, reaction)

	// Test: get reaction for nonexistent post
	reaction, err = store.GetUserReaction(ctx, "nonexistent-id", "user2")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, reaction)
}

// TestGetReactedUsers tests the GetReactedUsers method
func TestGetReactedUsers(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Add reactions from different users
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post.ID, "user3", ReactionLove)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post.ID, "user4", ReactionLike)
	assert.NoError(t, err)

	// Test: get all reacted users
	users, err := store.GetReactedUsers(ctx, post.ID, nil, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(users))
	assert.Contains(t, users, "user2")
	assert.Contains(t, users, "user3")
	assert.Contains(t, users, "user4")

	// Test: filter by reaction type
	likeReaction := ReactionLike
	users, err = store.GetReactedUsers(ctx, post.ID, &likeReaction, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))
	assert.Contains(t, users, "user2")
	assert.Contains(t, users, "user4")

	loveReaction := ReactionLove
	users, err = store.GetReactedUsers(ctx, post.ID, &loveReaction, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))
	assert.Contains(t, users, "user3")

	// Test: pagination
	users, err = store.GetReactedUsers(ctx, post.ID, nil, 1, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))

	users, err = store.GetReactedUsers(ctx, post.ID, nil, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))

	// Test: nonexistent post
	users, err = store.GetReactedUsers(ctx, "nonexistent-id", nil, 10, 0)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, users)
}

// TestGetReactionCounts tests the GetReactionCounts method
func TestGetReactionCounts(t *testing.T) {
	store := setupTestStore()
	ctx := context.Background()

	// Create a test post
	post := createTestPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Add different reactions
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post.ID, "user3", ReactionLike)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post.ID, "user4", ReactionLove)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post.ID, "user5", ReactionHaha)
	assert.NoError(t, err)

	// Test: get reaction counts
	counts, err := store.GetReactionCounts(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, counts[ReactionLike])
	assert.Equal(t, 1, counts[ReactionLove])
	assert.Equal(t, 1, counts[ReactionHaha])
	assert.Equal(t, 0, counts[ReactionWow])

	// Test: nonexistent post
	counts, err = store.GetReactionCounts(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, counts)
}
