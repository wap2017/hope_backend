package dao

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Post represents the posts table structure
type Post struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	UserID       int64  `json:"user_id"`
	Content      string `json:"content"`
	ViewCount    int    `json:"view_count" gorm:"default:0"`
	LikeCount    int    `json:"like_count" gorm:"default:0"`
	CommentCount int    `json:"comment_count" gorm:"default:0"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	// Virtual fields, not stored in database
	Images   []PostImage  `json:"images" gorm:"-"`
	Liked    bool         `json:"liked" gorm:"-"`
	UserInfo *UserProfile `json:"user_info,omitempty" gorm:"-"`
}

// TableName specifies the table name for GORM
func (Post) TableName() string {
	return "posts"
}

// PostImage represents the post_images table structure
type PostImage struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	PostID       int64  `json:"post_id"`
	ImagePath    string `json:"image_path"`
	DisplayOrder int    `json:"display_order"`
	CreatedAt    int64  `json:"created_at"`
}

// TableName specifies the table name for GORM
func (PostImage) TableName() string {
	return "post_images"
}

// PostLike represents the post_likes table structure
type PostLike struct {
	ID        int64 `json:"id" gorm:"primaryKey"`
	PostID    int64 `json:"post_id"`
	UserID    int64 `json:"user_id"`
	CreatedAt int64 `json:"created_at"`
}

// TableName specifies the table name for GORM
func (PostLike) TableName() string {
	return "post_likes"
}

// PostDAO handles database operations for posts
type PostDAO struct {
	db *gorm.DB
}

// NewPostDAO creates a new PostDAO
func NewPostDAO(db *gorm.DB) *PostDAO {
	return &PostDAO{db: db}
}

// Create inserts a new post with images
func (dao *PostDAO) Create(post *Post, imagePaths []string) (int64, error) {
	// Set timestamps
	now := time.Now().UnixMilli()
	post.CreatedAt = now
	post.UpdatedAt = now

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	// Create the post
	if err := tx.Create(post).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// Create post images if provided
	if len(imagePaths) > 0 {
		images := make([]PostImage, 0, len(imagePaths))
		for i, path := range imagePaths {
			images = append(images, PostImage{
				PostID:       post.ID,
				ImagePath:    path,
				DisplayOrder: i,
				CreatedAt:    now,
			})
		}

		if err := tx.Create(&images).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		// Populate the images in the post object
		post.Images = images
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return post.ID, nil
}

// GetByID retrieves a post by its ID with images
func (dao *PostDAO) GetByID(id int64, currentUserID int64) (*Post, error) {
	var post Post
	err := dao.db.First(&post, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("post not found")
		}
		return nil, err
	}

	// Get post images
	var images []PostImage
	err = dao.db.Where("post_id = ?", id).Order("display_order").Find(&images).Error
	if err != nil {
		return nil, err
	}
	post.Images = images

	// Check if current user liked this post
	var count int64
	dao.db.Model(&PostLike{}).Where("post_id = ? AND user_id = ?", id, currentUserID).Count(&count)
	post.Liked = count > 0

	// Get user info
	userDAO := NewUserProfileDAO(dao.db)
	userProfile, err := userDAO.GetByID(post.UserID)
	if err == nil {
		post.UserInfo = userProfile
	}

	// Increment view count
	dao.db.Model(&post).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))

	return &post, nil
}

// ListPosts retrieves a list of posts with pagination
func (dao *PostDAO) ListPosts(page, pageSize int, userID int64, currentUserID int64) ([]Post, int64, error) {
	var posts []Post
	var total int64

	query := dao.db.Model(&Post{})

	// Filter by user ID if provided
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	// Count total records for pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and order
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts).Error
	if err != nil {
		return nil, 0, err
	}

	// Get images for each post
	for i := range posts {
		var images []PostImage
		err = dao.db.Where("post_id = ?", posts[i].ID).Order("display_order").Find(&images).Error
		if err != nil {
			return nil, 0, err
		}
		posts[i].Images = images

		// Check if current user liked this post
		var count int64
		dao.db.Model(&PostLike{}).Where("post_id = ? AND user_id = ?", posts[i].ID, currentUserID).Count(&count)
		posts[i].Liked = count > 0

		// Get user info
		userDAO := NewUserProfileDAO(dao.db)
		userProfile, err := userDAO.GetByID(posts[i].UserID)
		if err == nil {
			posts[i].UserInfo = userProfile
		}
	}

	return posts, total, nil
}

// Update updates an existing post
func (dao *PostDAO) Update(post *Post) error {
	post.UpdatedAt = time.Now().UnixMilli()

	result := dao.db.Model(post).Updates(map[string]interface{}{
		"content":    post.Content,
		"updated_at": post.UpdatedAt,
	})

	return result.Error
}

// Delete deletes a post and all related data
func (dao *PostDAO) Delete(id int64) error {
	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Delete post images
	if err := tx.Where("post_id = ?", id).Delete(&PostImage{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete post likes
	if err := tx.Where("post_id = ?", id).Delete(&PostLike{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete comments and comment likes (handled by CommentDAO)
	commentDAO := NewCommentDAO(tx)
	if err := commentDAO.DeleteAllForPost(id); err != nil {
		tx.Rollback()
		return err
	}

	// Delete the post
	if err := tx.Delete(&Post{ID: id}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// LikePost adds a like to a post
func (dao *PostDAO) LikePost(postID, userID int64) error {
	// Check if post exists
	var post Post
	err := dao.db.First(&post, postID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("post not found")
		}
		return err
	}

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Check if user already liked the post
	var count int64
	tx.Model(&PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count)
	if count > 0 {
		tx.Rollback()
		return errors.New("post already liked by user")
	}

	// Add the like
	like := PostLike{
		PostID:    postID,
		UserID:    userID,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := tx.Create(&like).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Increment post like count
	if err := tx.Model(&Post{}).Where("id = ?", postID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// UnlikePost removes a like from a post
func (dao *PostDAO) UnlikePost(postID, userID int64) error {
	// Check if post exists
	var post Post
	err := dao.db.First(&post, postID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("post not found")
		}
		return err
	}

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Check if user liked the post
	var count int64
	tx.Model(&PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count)
	if count == 0 {
		tx.Rollback()
		return errors.New("post not liked by user")
	}

	// Remove the like
	if err := tx.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&PostLike{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Decrement post like count
	if err := tx.Model(&Post{}).Where("id = ?", postID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}
