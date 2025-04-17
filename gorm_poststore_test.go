package postflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestGormStore creates a new GormPostStore with an in-memory SQLite database for testing
func setupTestGormStore(t *testing.T) (*GormPostStore, *gorm.DB) {
	// Generate a unique database name for each test to prevent test interference
	dbName := fmt.Sprintf("file:memdb%d?mode=memory&cache=shared", time.Now().UnixNano())

	// Use in-memory SQLite database for testing with a unique name
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)

	// Create the store
	store, err := NewGormPostStore(db)
	require.NoError(t, err)

	// Return both the store and the DB so we can close it later
	return store, db
}

// cleanupTestDB closes the database connection
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}

// createTestGormPost creates a test post with the given userID
func createTestGormPost(userID string) *Post {
	return &Post{
		ID:         uuid.New().String(),
		UserID:     userID,
		Content:    "Test GORM content",
		Tags:       []string{"test", "gorm", "golang"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Reactions:  make(map[ReactionType]int),
		Comments:   0,
		Shares:     0,
		Visibility: "public",
	}
}

// createTestGormPostWithMedia creates a test post with media items
func createTestGormPostWithMedia(userID string) *Post {
	post := createTestGormPost(userID)
	post.Media = []Media{
		{
			ID:           uuid.New().String(),
			Type:         MediaTypeImage,
			URL:          "https://example.com/image.jpg",
			ThumbnailURL: "https://example.com/thumbnail.jpg",
			Description:  "Test image",
			Width:        800,
			Height:       600,
			FileSize:     1024 * 1024, // 1MB
			FileName:     "image.jpg",
			MimeType:     "image/jpeg",
			CreatedAt:    time.Now(),
		},
	}
	return post
}

// TestGormPostStore_SavePost tests the SavePost method
func TestGormPostStore_SavePost(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Test creating a new post
	post := createTestGormPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test retrieving the post
	savedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, post.ID, savedPost.ID)
	assert.Equal(t, post.UserID, savedPost.UserID)
	assert.Equal(t, post.Content, savedPost.Content)
	assert.Equal(t, post.Visibility, savedPost.Visibility)

	// Test updating a post
	post.Content = "Updated GORM content"
	post.Tags = []string{"test", "gorm", "golang", "updated"}
	err = store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test retrieving the updated post
	updatedPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated GORM content", updatedPost.Content)
	assert.Equal(t, 4, len(updatedPost.Tags))
	assert.Contains(t, updatedPost.Tags, "updated")

	// Test creating a post with media
	postWithMedia := createTestGormPostWithMedia("user2")
	err = store.SavePost(ctx, postWithMedia)
	assert.NoError(t, err)

	// Test retrieving the post with media
	savedPostWithMedia, err := store.GetPost(ctx, postWithMedia.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(savedPostWithMedia.Media))
	assert.Equal(t, postWithMedia.Media[0].URL, savedPostWithMedia.Media[0].URL)
	assert.Equal(t, postWithMedia.Media[0].Type, savedPostWithMedia.Media[0].Type)

	// Test updating post with media (replacing media)
	postWithMedia.Media[0].Description = "Updated media description"
	err = store.SavePost(ctx, postWithMedia)
	assert.NoError(t, err)

	updatedPostWithMedia, err := store.GetPost(ctx, postWithMedia.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated media description", updatedPostWithMedia.Media[0].Description)
}

// TestGormPostStore_GetPost tests the GetPost method
func TestGormPostStore_GetPost(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test getting a post that exists
	foundPost, err := store.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	assert.Equal(t, post.ID, foundPost.ID)
	assert.Equal(t, post.UserID, foundPost.UserID)

	// Test getting a post that doesn't exist
	_, err = store.GetPost(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestGormPostStore_DeletePost tests the DeletePost method
func TestGormPostStore_DeletePost(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
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

// TestGormPostStore_ListPosts tests the ListPosts method
func TestGormPostStore_ListPosts(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create test posts for two users
	for i := 0; i < 5; i++ {
		post := createTestGormPost("user1")
		post.Content = "User1 content " + uuid.New().String()
		err := store.SavePost(ctx, post)
		assert.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		post := createTestGormPost("user2")
		post.Content = "User2 content " + uuid.New().String()
		err := store.SavePost(ctx, post)
		assert.NoError(t, err)
	}

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
		Tags: []string{"gorm"},
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
	if len(posts) > 1 {
		assert.True(t, posts[0].CreatedAt.Before(posts[len(posts)-1].CreatedAt) ||
			posts[0].CreatedAt.Equal(posts[len(posts)-1].CreatedAt))
	}

	// Test: list posts with sorting (created_at, desc)
	filter.SortOrder = "desc"
	posts, err = store.ListPosts(ctx, filter)
	assert.NoError(t, err)
	if len(posts) > 1 {
		assert.True(t, posts[0].CreatedAt.After(posts[len(posts)-1].CreatedAt) ||
			posts[0].CreatedAt.Equal(posts[len(posts)-1].CreatedAt))
	}

	// Test: list posts with visibility filter
	// First, create a private post
	privatePost := createTestGormPost("user1")
	privatePost.Visibility = "private"
	err = store.SavePost(ctx, privatePost)
	assert.NoError(t, err)

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

// TestGormPostStore_GetUserFeed tests the GetUserFeed method
func TestGormPostStore_GetUserFeed(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create test posts, some public, some private
	for i := 0; i < 5; i++ {
		post := createTestGormPost("user1")
		post.Visibility = "public"
		err := store.SavePost(ctx, post)
		assert.NoError(t, err)
	}
	for i := 0; i < 3; i++ {
		post := createTestGormPost("user2")
		post.Visibility = "private"
		err := store.SavePost(ctx, post)
		assert.NoError(t, err)
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

// TestGormPostStore_GetTrendingPosts tests the GetTrendingPosts method
func TestGormPostStore_GetTrendingPosts(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create test posts with different engagement levels
	post1 := createTestGormPost("user1")
	post1.Comments = 7
	post1.Shares = 3
	err := store.SavePost(ctx, post1)
	assert.NoError(t, err)

	post2 := createTestGormPost("user1")
	post2.Comments = 3
	post2.Shares = 1
	err = store.SavePost(ctx, post2)
	assert.NoError(t, err)

	post3 := createTestGormPost("user2")
	post3.Comments = 12
	post3.Shares = 5
	err = store.SavePost(ctx, post3)
	assert.NoError(t, err)

	// Add reactions to posts
	err = store.SaveReaction(ctx, post1.ID, "user2", ReactionLike)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post1.ID, "user3", ReactionLove)
	assert.NoError(t, err)

	err = store.SaveReaction(ctx, post2.ID, "user3", ReactionLike)
	assert.NoError(t, err)

	err = store.SaveReaction(ctx, post3.ID, "user1", ReactionLike)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post3.ID, "user4", ReactionWow)
	assert.NoError(t, err)
	err = store.SaveReaction(ctx, post3.ID, "user5", ReactionLike)
	assert.NoError(t, err)

	// Test: get trending posts
	posts, err := store.GetTrendingPosts(ctx, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts))

	// The most engaging post should be first (post3)
	assert.Equal(t, post3.ID, posts[0].ID)

	// Test: limit
	allPosts, err := store.GetTrendingPosts(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(allPosts))
}

// TestGormPostStore_SaveReaction tests the SaveReaction method
func TestGormPostStore_SaveReaction(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Test: save a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction was saved
	reaction, err := store.GetUserReaction(ctx, post.ID, "user2")
	assert.NoError(t, err)
	assert.NotNil(t, reaction)
	assert.Equal(t, ReactionLike, *reaction)

	// Test: change a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLove)
	assert.NoError(t, err)

	// Verify reaction was updated
	reaction, err = store.GetUserReaction(ctx, post.ID, "user2")
	assert.NoError(t, err)
	assert.NotNil(t, reaction)
	assert.Equal(t, ReactionLove, *reaction)

	// Test: invalid reaction type
	err = store.SaveReaction(ctx, post.ID, "user3", ReactionType(100))
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidReaction, err)

	// Test: reaction to nonexistent post
	err = store.SaveReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestGormPostStore_DeleteReaction tests the DeleteReaction method
func TestGormPostStore_DeleteReaction(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
	err := store.SavePost(ctx, post)
	assert.NoError(t, err)

	// Add a reaction
	err = store.SaveReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: delete the reaction
	err = store.DeleteReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Verify reaction was removed
	reaction, err := store.GetUserReaction(ctx, post.ID, "user2")
	assert.NoError(t, err)
	assert.Nil(t, reaction)

	// Test: delete a nonexistent reaction (should not error)
	err = store.DeleteReaction(ctx, post.ID, "user2", ReactionLike)
	assert.NoError(t, err)

	// Test: delete reaction for nonexistent post
	err = store.DeleteReaction(ctx, "nonexistent-id", "user2", ReactionLike)
	assert.Error(t, err)
	assert.Equal(t, ErrPostNotFound, err)
}

// TestGormPostStore_GetUserReaction tests the GetUserReaction method
func TestGormPostStore_GetUserReaction(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
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

// TestGormPostStore_GetReactedUsers tests the GetReactedUsers method
func TestGormPostStore_GetReactedUsers(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
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

// TestGormPostStore_GetReactionCounts tests the GetReactionCounts method
func TestGormPostStore_GetReactionCounts(t *testing.T) {
	store, db := setupTestGormStore(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create a test post
	post := createTestGormPost("user1")
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
