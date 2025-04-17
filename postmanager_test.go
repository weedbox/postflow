package postflow

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// setupTestPostManager creates a new PostManagerImpl with an InMemoryPostStore for testing
func setupTestPostManager() *PostManagerImpl {
	store := NewInMemoryPostStore()
	return NewPostManager(store)
}

// createTestPostData creates a test post with the given userID
func createTestPostData(userID string) *Post {
	return &Post{
		ID:         uuid.New().String(),
		UserID:     userID,
		Content:    "Test content for PostManager",
		Tags:       []string{"test", "golang", "postmanager"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Reactions:  make(map[ReactionType]int),
		Comments:   0,
		Shares:     0,
		Visibility: "public",
	}
}

// TestPostManagerCreatePost tests the CreatePost method
func TestPostManagerCreatePost(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Test: create a post with valid data
	post := createTestPostData("user1")
	post.ID = "" // Let the manager generate an ID
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)
	assert.NotEmpty(t, postID)
	assert.Equal(t, postID, post.ID) // Check that the ID was set in the post

	// Verify the post was created
	createdPost, err := pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, postID, createdPost.ID)
	assert.Equal(t, "user1", createdPost.UserID)
	assert.Equal(t, "Test content for PostManager", createdPost.Content)

	// Test: create a post without a user ID (should fail)
	invalidPost := createTestPostData("")
	_, err = pm.CreatePost(ctx, invalidPost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

// TestPostManagerGetPost tests the GetPost method
func TestPostManagerGetPost(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Test: get a post that exists
	foundPost, err := pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, postID, foundPost.ID)
	assert.Equal(t, "user1", foundPost.UserID)
	assert.Equal(t, "Test content for PostManager", foundPost.Content)

	// Test: get a post that doesn't exist
	_, err = pm.GetPost(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestPostManagerUpdatePost tests the UpdatePost method
func TestPostManagerUpdatePost(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Test: update the post with valid changes
	post.Content = "Updated content for PostManager"
	post.Tags = []string{"test", "golang", "postmanager", "updated"}
	err = pm.UpdatePost(ctx, post)
	assert.NoError(t, err)

	// Verify the post was updated
	updatedPost, err := pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated content for PostManager", updatedPost.Content)
	assert.Equal(t, 4, len(updatedPost.Tags))
	assert.Contains(t, updatedPost.Tags, "updated")

	// Test: update a post with a different user (should fail)
	unauthorizedPost := createTestPostData("user2")
	unauthorizedPost.ID = postID
	err = pm.UpdatePost(ctx, unauthorizedPost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized to update this post")

	// Test: update a post that doesn't exist
	nonexistentPost := createTestPostData("user1")
	nonexistentPost.ID = "nonexistent-id"
	err = pm.UpdatePost(ctx, nonexistentPost)
	assert.Error(t, err)

	// Test: update a post without an ID (should fail)
	invalidPost := createTestPostData("user1")
	invalidPost.ID = ""
	err = pm.UpdatePost(ctx, invalidPost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post ID is required")
}

// TestPostManagerDeletePost tests the DeletePost method
func TestPostManagerDeletePost(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Test: delete by a different user (should fail)
	err = pm.DeletePost(ctx, postID, "user2")
	assert.Error(t, err)
	assert.Equal(t, ErrPermissionDenied, err)

	// Verify post still exists
	_, err = pm.GetPost(ctx, postID)
	assert.NoError(t, err)

	// Test: delete by correct user
	err = pm.DeletePost(ctx, postID, "user1")
	assert.NoError(t, err)

	// Verify post is deleted
	_, err = pm.GetPost(ctx, postID)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)

	// Test: delete a nonexistent post
	err = pm.DeletePost(ctx, "nonexistent-id", "user1")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestPostManagerListPosts tests the ListPosts method
func TestPostManagerListPosts(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create test posts for two users
	for i := 0; i < 5; i++ {
		post := createTestPostData("user1")
		post.Content = "User1 content " + uuid.New().String()
		_, err := pm.CreatePost(ctx, post)
		assert.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		post := createTestPostData("user2")
		post.Content = "User2 content " + uuid.New().String()
		_, err := pm.CreatePost(ctx, post)
		assert.NoError(t, err)
	}

	// Test: list posts by user1
	filter := &PostFilter{
		UserID: "user1",
	}
	posts, err := pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(posts))
	for _, post := range posts {
		assert.Equal(t, "user1", post.UserID)
	}

	// Test: list posts by user2
	filter.UserID = "user2"
	posts, err = pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	// Test: list posts with tag filter
	filter = &PostFilter{
		Tags: []string{"golang"},
	}
	posts, err = pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(posts))

	// Test: list posts with pagination
	filter = &PostFilter{
		Limit:  3,
		Offset: 0,
	}
	posts, err = pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	// Test: list posts with sorting (created_at, desc)
	filter = &PostFilter{
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	posts, err = pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.True(t, posts[0].CreatedAt.Equal(posts[0].CreatedAt) ||
		posts[0].CreatedAt.After(posts[len(posts)-1].CreatedAt))

	// Test: list posts with visibility filter
	privatePost := createTestPostData("user1")
	privatePost.Visibility = "private"
	_, err = pm.CreatePost(ctx, privatePost)
	assert.NoError(t, err)

	filter = &PostFilter{
		Visibility: "private",
	}
	posts, err = pm.ListPosts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(posts))
	assert.Equal(t, "private", posts[0].Visibility)
}

// TestPostManagerGetUserFeed tests the GetUserFeed method
func TestPostManagerGetUserFeed(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create test posts, some public, some private
	for i := 0; i < 5; i++ {
		post := createTestPostData("user1")
		post.Visibility = "public"
		_, err := pm.CreatePost(ctx, post)
		assert.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		post := createTestPostData("user2")
		post.Visibility = "private"
		_, err := pm.CreatePost(ctx, post)
		assert.NoError(t, err)
	}

	// Test: get user feed for user3 (should only see public posts)
	posts, err := pm.GetUserFeed(ctx, "user3", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(posts))
	for _, post := range posts {
		assert.Equal(t, "public", post.Visibility)
	}

	// Test: pagination
	postsPage1, err := pm.GetUserFeed(ctx, "user3", 2, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(postsPage1))

	postsPage2, err := pm.GetUserFeed(ctx, "user3", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(postsPage2))

	// Ensure pages are different
	assert.NotEqual(t, postsPage1[0].ID, postsPage2[0].ID)
}

// TestPostManagerGetTrendingPosts tests the GetTrendingPosts method
func TestPostManagerGetTrendingPosts(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create test posts with different engagement levels
	post1 := createTestPostData("user1")
	post1.Reactions = map[ReactionType]int{ReactionLike: 10, ReactionLove: 5}
	post1.Comments = 7
	post1.Shares = 3
	_, err := pm.CreatePost(ctx, post1)
	assert.NoError(t, err)

	post2 := createTestPostData("user1")
	post2.Reactions = map[ReactionType]int{ReactionLike: 5, ReactionHaha: 2}
	post2.Comments = 3
	post2.Shares = 1
	_, err = pm.CreatePost(ctx, post2)
	assert.NoError(t, err)

	post3 := createTestPostData("user2")
	post3.Reactions = map[ReactionType]int{ReactionLike: 20, ReactionWow: 8}
	post3.Comments = 12
	post3.Shares = 5
	postID3, err := pm.CreatePost(ctx, post3)
	assert.NoError(t, err)

	// Test: get trending posts
	posts, err := pm.GetTrendingPosts(ctx, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts))

	// The most engaging post should be first
	assert.Equal(t, postID3, posts[0].ID)

	// Test: limit
	allPosts, err := pm.GetTrendingPosts(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(allPosts))
}

// TestPostManagerAddReaction tests the AddReaction method
func TestPostManagerAddReaction(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Test: add a reaction
	err = pm.AddReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction count was updated
	updatedPost, err := pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedPost.Reactions[ReactionLike])

	// Test: change a reaction
	err = pm.AddReaction(ctx, postID, "user2", ReactionLove)
	assert.NoError(t, err)

	// Verify reaction counts were updated
	updatedPost, err = pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, 0, updatedPost.Reactions[ReactionLike])
	assert.Equal(t, 1, updatedPost.Reactions[ReactionLove])

	// Test: invalid reaction type
	err = pm.AddReaction(ctx, postID, "user3", ReactionType(100))
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidReaction, err)

	// Test: reaction to nonexistent post
	err = pm.AddReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestPostManagerRemoveReaction tests the RemoveReaction method
func TestPostManagerRemoveReaction(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Add a reaction
	err = pm.AddReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: remove the reaction
	err = pm.RemoveReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction was removed
	updatedPost, err := pm.GetPost(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, 0, updatedPost.Reactions[ReactionLike])

	// Test: remove a nonexistent reaction (should not error)
	err = pm.RemoveReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: remove reaction for nonexistent post
	err = pm.RemoveReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestPostManagerGetUserReaction tests the GetUserReaction method
func TestPostManagerGetUserReaction(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Add a reaction
	err = pm.AddReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: get user reaction that exists
	reaction, err := pm.GetUserReaction(ctx, postID, "user2")
	assert.NoError(t, err)
	assert.NotNil(t, reaction)
	assert.Equal(t, ReactionLike, *reaction)

	// Test: get user reaction that doesn't exist
	reaction, err = pm.GetUserReaction(ctx, postID, "user3")
	assert.NoError(t, err)
	assert.Nil(t, reaction)

	// Test: get reaction for nonexistent post
	reaction, err = pm.GetUserReaction(ctx, "nonexistent-id", "user2")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, reaction)
}

// TestPostManagerGetReactedUsers tests the GetReactedUsers method
func TestPostManagerGetReactedUsers(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Add reactions from different users
	err = pm.AddReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)
	err = pm.AddReaction(ctx, postID, "user3", ReactionLove)
	assert.NoError(t, err)
	err = pm.AddReaction(ctx, postID, "user4", ReactionLike)
	assert.NoError(t, err)

	// Test: get all reacted users
	users, err := pm.GetReactedUsers(ctx, postID, nil, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(users))
	assert.Contains(t, users, "user2")
	assert.Contains(t, users, "user3")
	assert.Contains(t, users, "user4")

	// Test: filter by reaction type
	likeReaction := ReactionLike
	users, err = pm.GetReactedUsers(ctx, postID, &likeReaction, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))
	assert.Contains(t, users, "user2")
	assert.Contains(t, users, "user4")

	loveReaction := ReactionLove
	users, err = pm.GetReactedUsers(ctx, postID, &loveReaction, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))
	assert.Contains(t, users, "user3")

	// Test: pagination
	users, err = pm.GetReactedUsers(ctx, postID, nil, 1, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))

	users, err = pm.GetReactedUsers(ctx, postID, nil, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))

	// Test: nonexistent post
	users, err = pm.GetReactedUsers(ctx, "nonexistent-id", nil, 10, 0)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, users)
}

// TestPostManagerGetReactionCounts tests the GetReactionCounts method
func TestPostManagerGetReactionCounts(t *testing.T) {
	pm := setupTestPostManager()
	ctx := context.Background()

	// Create a test post
	post := createTestPostData("user1")
	postID, err := pm.CreatePost(ctx, post)
	assert.NoError(t, err)

	// Add different reactions
	err = pm.AddReaction(ctx, postID, "user2", ReactionLike)
	assert.NoError(t, err)
	err = pm.AddReaction(ctx, postID, "user3", ReactionLike)
	assert.NoError(t, err)
	err = pm.AddReaction(ctx, postID, "user4", ReactionLove)
	assert.NoError(t, err)
	err = pm.AddReaction(ctx, postID, "user5", ReactionHaha)
	assert.NoError(t, err)

	// Test: get reaction counts
	counts, err := pm.GetReactionCounts(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, 2, counts[ReactionLike])
	assert.Equal(t, 1, counts[ReactionLove])
	assert.Equal(t, 1, counts[ReactionHaha])
	assert.Equal(t, 0, counts[ReactionWow])

	// Test: nonexistent post
	counts, err = pm.GetReactionCounts(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
	assert.Nil(t, counts)
}
