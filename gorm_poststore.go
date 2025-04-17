// Package postflow provides functionality for managing user posts and feeds.
package postflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GormPostStore implements PostStore interface with GORM as the underlying storage
type GormPostStore struct {
	db *gorm.DB
}

// PostModel is the GORM model for storing posts
type PostModel struct {
	ID         string `gorm:"primaryKey"`
	UserID     string `gorm:"index"`
	Content    string
	Media      []MediaModel `gorm:"foreignKey:PostID"`
	Tags       []TagModel   `gorm:"many2many:post_tags;"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Visibility string
	Comments   int
	Shares     int
	// Reactions will be stored in a separate table
}

// MediaModel is the GORM model for storing media items
type MediaModel struct {
	ID           string `gorm:"primaryKey"`
	PostID       string `gorm:"index"`
	Type         uint8
	URL          string
	ThumbnailURL string
	Description  string
	Width        int
	Height       int
	Duration     int
	FileSize     int64
	FileName     string
	MimeType     string
	CreatedAt    time.Time
}

// TagModel is the GORM model for storing tags
type TagModel struct {
	Name string `gorm:"primaryKey"`
}

// ReactionModel is the GORM model for storing user reactions to posts
type ReactionModel struct {
	PostID       string `gorm:"primaryKey;index"`
	UserID       string `gorm:"primaryKey;index"`
	ReactionType uint8
	CreatedAt    time.Time
}

// NewGormPostStore creates a new instance of GormPostStore
func NewGormPostStore(db *gorm.DB) (*GormPostStore, error) {
	// Auto-migrate the models to ensure tables exist
	err := db.AutoMigrate(&PostModel{}, &MediaModel{}, &TagModel{}, &ReactionModel{})
	if err != nil {
		return nil, err
	}

	return &GormPostStore{
		db: db,
	}, nil
}

// Convert PostModel to Post
func (s *GormPostStore) toPost(postModel *PostModel, reactions map[ReactionType]int) *Post {
	post := &Post{
		ID:         postModel.ID,
		UserID:     postModel.UserID,
		Content:    postModel.Content,
		CreatedAt:  postModel.CreatedAt,
		UpdatedAt:  postModel.UpdatedAt,
		Reactions:  reactions,
		Comments:   postModel.Comments,
		Shares:     postModel.Shares,
		Visibility: postModel.Visibility,
	}

	// Convert MediaModel to Media
	if len(postModel.Media) > 0 {
		post.Media = make([]Media, len(postModel.Media))
		for i, mediaModel := range postModel.Media {
			post.Media[i] = Media{
				ID:           mediaModel.ID,
				Type:         MediaType(mediaModel.Type),
				URL:          mediaModel.URL,
				ThumbnailURL: mediaModel.ThumbnailURL,
				Description:  mediaModel.Description,
				Width:        mediaModel.Width,
				Height:       mediaModel.Height,
				Duration:     mediaModel.Duration,
				FileSize:     mediaModel.FileSize,
				FileName:     mediaModel.FileName,
				MimeType:     mediaModel.MimeType,
				CreatedAt:    mediaModel.CreatedAt,
			}
		}
	}

	// Convert TagModel to string tags
	if len(postModel.Tags) > 0 {
		post.Tags = make([]string, len(postModel.Tags))
		for i, tagModel := range postModel.Tags {
			post.Tags[i] = tagModel.Name
		}
	}

	return post
}

// SavePost saves a new post or updates an existing post
func (s *GormPostStore) SavePost(ctx context.Context, post *Post) error {
	// Use transaction to ensure data consistency
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingPost PostModel

		// Check if the post already exists
		err := tx.Where("id = ?", post.ID).First(&existingPost).Error
		isNew := errors.Is(err, gorm.ErrRecordNotFound)

		if isNew {
			// Create new post
			postModel := PostModel{
				ID:         post.ID,
				UserID:     post.UserID,
				Content:    post.Content,
				CreatedAt:  post.CreatedAt,
				UpdatedAt:  post.UpdatedAt,
				Visibility: post.Visibility,
				Comments:   post.Comments,
				Shares:     post.Shares,
			}

			if err := tx.Create(&postModel).Error; err != nil {
				return err
			}

			// Create media entries
			if len(post.Media) > 0 {
				mediaModels := make([]MediaModel, len(post.Media))
				for i, media := range post.Media {
					mediaModels[i] = MediaModel{
						ID:           media.ID,
						PostID:       post.ID,
						Type:         uint8(media.Type),
						URL:          media.URL,
						ThumbnailURL: media.ThumbnailURL,
						Description:  media.Description,
						Width:        media.Width,
						Height:       media.Height,
						Duration:     media.Duration,
						FileSize:     media.FileSize,
						FileName:     media.FileName,
						MimeType:     media.MimeType,
						CreatedAt:    media.CreatedAt,
					}
				}
				if err := tx.Create(&mediaModels).Error; err != nil {
					return err
				}
			}

			// Create tag associations
			if len(post.Tags) > 0 {
				// First ensure all tags exist
				for _, tag := range post.Tags {
					var tagModel TagModel
					if err := tx.Where("name = ?", tag).FirstOrCreate(&tagModel, TagModel{Name: tag}).Error; err != nil {
						return err
					}
				}

				// Then associate tags with post
				var tags []TagModel
				for _, tag := range post.Tags {
					tags = append(tags, TagModel{Name: tag})
				}
				if err := tx.Model(&postModel).Association("Tags").Replace(tags); err != nil {
					return err
				}
			}

			// Create reaction counts if any
			if len(post.Reactions) > 0 {
				for reactionType, count := range post.Reactions {
					// This is a placeholder - in a real system, you would need to create individual reactions
					// Here we're just simulating the reaction counts for a new post
					for i := 0; i < count; i++ {
						reactionModel := ReactionModel{
							PostID:       post.ID,
							UserID:       fmt.Sprintf("system-%d", i), // Fixed: Proper conversion of int to string
							ReactionType: uint8(reactionType),
							CreatedAt:    time.Now(),
						}
						if err := tx.Create(&reactionModel).Error; err != nil {
							return err
						}
					}
				}
			}
		} else {
			// Update existing post
			existingPost.UserID = post.UserID
			existingPost.Content = post.Content
			existingPost.UpdatedAt = post.UpdatedAt
			existingPost.Visibility = post.Visibility
			existingPost.Comments = post.Comments
			existingPost.Shares = post.Shares

			if err := tx.Save(&existingPost).Error; err != nil {
				return err
			}

			// Update media: delete existing and create new
			if err := tx.Where("post_id = ?", post.ID).Delete(&MediaModel{}).Error; err != nil {
				return err
			}

			if len(post.Media) > 0 {
				mediaModels := make([]MediaModel, len(post.Media))
				for i, media := range post.Media {
					mediaModels[i] = MediaModel{
						ID:           media.ID,
						PostID:       post.ID,
						Type:         uint8(media.Type),
						URL:          media.URL,
						ThumbnailURL: media.ThumbnailURL,
						Description:  media.Description,
						Width:        media.Width,
						Height:       media.Height,
						Duration:     media.Duration,
						FileSize:     media.FileSize,
						FileName:     media.FileName,
						MimeType:     media.MimeType,
						CreatedAt:    media.CreatedAt,
					}
				}
				if err := tx.Create(&mediaModels).Error; err != nil {
					return err
				}
			}

			// Update tags
			var tags []TagModel
			for _, tag := range post.Tags {
				// Ensure the tag exists
				var tagModel TagModel
				if err := tx.Where("name = ?", tag).FirstOrCreate(&tagModel, TagModel{Name: tag}).Error; err != nil {
					return err
				}
				tags = append(tags, tagModel)
			}

			// Replace all tags associations
			if err := tx.Model(&existingPost).Association("Tags").Replace(tags); err != nil {
				return err
			}

			// Note: We don't update reactions here as they are managed separately via SaveReaction
		}

		return nil
	})
}

// GetPost retrieves a post by its ID
func (s *GormPostStore) GetPost(ctx context.Context, postID string) (*Post, error) {
	// Get post with preloaded associations
	var postModel PostModel
	err := s.db.WithContext(ctx).
		Preload("Media").
		Preload("Tags").
		Where("id = ?", postID).
		First(&postModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	// Get reaction counts
	reactionCounts, err := s.GetReactionCounts(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Convert model to domain object
	return s.toPost(&postModel, reactionCounts), nil
}

// DeletePost removes a post from the store
func (s *GormPostStore) DeletePost(ctx context.Context, postID string, userID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if post exists and belongs to the user
		var postModel PostModel
		if err := tx.Where("id = ? AND user_id = ?", postID, userID).First(&postModel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Either post doesn't exist or user doesn't own it, check which one
				var count int64
				if err := tx.Model(&PostModel{}).Where("id = ?", postID).Count(&count).Error; err != nil {
					return err
				}

				if count == 0 {
					return ErrPostNotFound
				}
				return ErrPermissionDenied
			}
			return err
		}

		// Delete reactions
		if err := tx.Where("post_id = ?", postID).Delete(&ReactionModel{}).Error; err != nil {
			return err
		}

		// Delete media
		if err := tx.Where("post_id = ?", postID).Delete(&MediaModel{}).Error; err != nil {
			return err
		}

		// Remove tag associations
		if err := tx.Model(&postModel).Association("Tags").Clear(); err != nil {
			return err
		}

		// Delete the post
		if err := tx.Delete(&postModel).Error; err != nil {
			return err
		}

		return nil
	})
}

// ListPosts retrieves posts based on filter criteria
func (s *GormPostStore) ListPosts(ctx context.Context, filter *PostFilter) ([]*Post, error) {
	query := s.db.WithContext(ctx).
		Model(&PostModel{}).
		Preload("Media").
		Preload("Tags")

	// Apply filters
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}

	if filter.Visibility != "" {
		query = query.Where("visibility = ?", filter.Visibility)
	}

	if filter.TimeRange != nil {
		query = query.Where("created_at BETWEEN ? AND ?", filter.TimeRange.Start, filter.TimeRange.End)
	}

	// Apply tag filters if any
	if len(filter.Tags) > 0 {
		// Find posts with ALL the specified tags
		for _, tag := range filter.Tags {
			// Create a subquery for each tag
			// The join table is post_tags with post_model_id and tag_model_name columns
			query = query.Where("id IN (SELECT post_model_id FROM post_tags WHERE tag_model_name = ?)", tag)
		}
	}

	// Apply sorting
	if filter.SortBy != "" {
		// Map the sort field to database column
		sortField := filter.SortBy
		switch sortField {
		case "created_at", "updated_at", "comments", "shares":
			// These fields exist directly in the posts table
			break
		case "reactions":
			// For sorting by reactions, we need to count reactions in a subquery
			// This is complex and might impact performance, so we'll use a simple approach here
			// In a real system, you might want to denormalize this or use a more efficient approach
			sortField = "created_at" // Fallback to created_at
		default:
			sortField = "created_at" // Default sort
		}

		// Apply sort order
		sortOrder := "DESC"
		if filter.SortOrder == "asc" {
			sortOrder = "ASC"
		}

		query = query.Order(sortField + " " + sortOrder)
	} else {
		// Default sorting by created_at desc
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	// Execute query
	var postModels []PostModel
	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain objects
	posts := make([]*Post, len(postModels))
	for i, postModel := range postModels {
		// Get reaction counts for each post
		reactionCounts, err := s.GetReactionCounts(ctx, postModel.ID)
		if err != nil {
			return nil, err
		}

		posts[i] = s.toPost(&postModel, reactionCounts)
	}

	return posts, nil
}

// GetUserFeed retrieves posts for a user's feed
func (s *GormPostStore) GetUserFeed(ctx context.Context, userID string, limit, offset int) ([]*Post, error) {
	// In a real implementation, this would consider followed users, algorithms, etc.
	// This simple version just returns recent public posts

	// Get public posts sorted by creation time, newest first
	query := s.db.WithContext(ctx).
		Model(&PostModel{}).
		Preload("Media").
		Preload("Tags").
		Where("visibility = ?", "public").
		Order("created_at DESC")

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	// Execute query
	var postModels []PostModel
	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain objects
	posts := make([]*Post, len(postModels))
	for i, postModel := range postModels {
		// Get reaction counts for each post
		reactionCounts, err := s.GetReactionCounts(ctx, postModel.ID)
		if err != nil {
			return nil, err
		}

		posts[i] = s.toPost(&postModel, reactionCounts)
	}

	return posts, nil
}

// GetTrendingPosts retrieves currently trending posts
func (s *GormPostStore) GetTrendingPosts(ctx context.Context, limit int) ([]*Post, error) {
	// In a real system, this would be more complex, possibly using a scoring algorithm
	// For simplicity, we'll get posts with the most reactions + comments + shares

	// Query to get public posts with reaction, comment, and share counts
	query := s.db.WithContext(ctx).
		Model(&PostModel{}).
		Preload("Media").
		Preload("Tags").
		Joins("LEFT JOIN (SELECT post_id, COUNT(*) as reaction_count FROM reaction_models GROUP BY post_id) r ON post_models.id = r.post_id").
		Where("visibility = ?", "public").
		Select("post_models.*, COALESCE(r.reaction_count, 0) + post_models.comments + post_models.shares as engagement").
		Order("engagement DESC")

	// Apply limit
	if limit > 0 {
		query = query.Limit(limit)
	}

	// Execute query
	var postModels []PostModel
	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain objects
	posts := make([]*Post, len(postModels))
	for i, postModel := range postModels {
		// Get reaction counts for each post
		reactionCounts, err := s.GetReactionCounts(ctx, postModel.ID)
		if err != nil {
			return nil, err
		}

		posts[i] = s.toPost(&postModel, reactionCounts)
	}

	return posts, nil
}

// SaveReaction saves a reaction to a post
func (s *GormPostStore) SaveReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	// Check if post exists
	var count int64
	if err := s.db.WithContext(ctx).Model(&PostModel{}).Where("id = ?", postID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrPostNotFound
	}

	// Check if reaction type is valid
	if reactionType <= ReactionNone || reactionType > ReactionAngry {
		return ErrInvalidReaction
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if user already has a reaction to this post
		var existingReaction ReactionModel
		err := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existingReaction).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new reaction
			newReaction := ReactionModel{
				PostID:       postID,
				UserID:       userID,
				ReactionType: uint8(reactionType),
				CreatedAt:    time.Now(),
			}

			if err := tx.Create(&newReaction).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			// Update existing reaction if different
			if uint8(reactionType) != existingReaction.ReactionType {
				existingReaction.ReactionType = uint8(reactionType)
				existingReaction.CreatedAt = time.Now()

				if err := tx.Save(&existingReaction).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// DeleteReaction removes a reaction from a post
func (s *GormPostStore) DeleteReaction(ctx context.Context, postID string, userID string, reactionType ReactionType) error {
	// Check if post exists
	var count int64
	if err := s.db.WithContext(ctx).Model(&PostModel{}).Where("id = ?", postID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrPostNotFound
	}

	// Delete the reaction
	return s.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ? AND reaction_type = ?", postID, userID, uint8(reactionType)).
		Delete(&ReactionModel{}).Error
}

// GetUserReaction gets the current reaction of a user for a post
func (s *GormPostStore) GetUserReaction(ctx context.Context, postID string, userID string) (*ReactionType, error) {
	// Check if post exists
	var postCount int64
	if err := s.db.WithContext(ctx).Model(&PostModel{}).Where("id = ?", postID).Count(&postCount).Error; err != nil {
		return nil, err
	}
	if postCount == 0 {
		return nil, ErrPostNotFound
	}

	// Get user's reaction
	var reaction ReactionModel
	err := s.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		First(&reaction).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // User hasn't reacted
	} else if err != nil {
		return nil, err
	}

	// Convert to ReactionType
	reactionType := ReactionType(reaction.ReactionType)
	return &reactionType, nil
}

// GetReactedUsers returns users who reacted to a specific post
func (s *GormPostStore) GetReactedUsers(ctx context.Context, postID string, reactionType *ReactionType, limit, offset int) ([]string, error) {
	// Check if post exists
	var postCount int64
	if err := s.db.WithContext(ctx).Model(&PostModel{}).Where("id = ?", postID).Count(&postCount).Error; err != nil {
		return nil, err
	}
	if postCount == 0 {
		return nil, ErrPostNotFound
	}

	// Build query
	query := s.db.WithContext(ctx).
		Model(&ReactionModel{}).
		Where("post_id = ?", postID)

	// Filter by reaction type if specified
	if reactionType != nil {
		query = query.Where("reaction_type = ?", uint8(*reactionType))
	}

	// Order by most recent first
	query = query.Order("created_at DESC")

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	// Execute query
	var reactions []ReactionModel
	if err := query.Find(&reactions).Error; err != nil {
		return nil, err
	}

	// Extract user IDs
	userIDs := make([]string, len(reactions))
	for i, reaction := range reactions {
		userIDs[i] = reaction.UserID
	}

	return userIDs, nil
}

// GetReactionCounts returns the count of each reaction type for a post
func (s *GormPostStore) GetReactionCounts(ctx context.Context, postID string) (map[ReactionType]int, error) {
	// Check if post exists
	var postCount int64
	if err := s.db.WithContext(ctx).Model(&PostModel{}).Where("id = ?", postID).Count(&postCount).Error; err != nil {
		return nil, err
	}
	if postCount == 0 {
		return nil, ErrPostNotFound
	}

	// Get counts grouped by reaction type
	type Result struct {
		ReactionType uint8
		Count        int
	}
	var results []Result

	err := s.db.WithContext(ctx).
		Model(&ReactionModel{}).
		Select("reaction_type, count(*) as count").
		Where("post_id = ?", postID).
		Group("reaction_type").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to map
	counts := make(map[ReactionType]int)
	for _, result := range results {
		counts[ReactionType(result.ReactionType)] = result.Count
	}

	return counts, nil
}
