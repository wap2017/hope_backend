package dao

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Comment represents the comments table structure
type Comment struct {
	ID         int64  `json:"id" gorm:"primaryKey"`
	PostID     int64  `json:"post_id"`
	UserID     int64  `json:"user_id"`
	ParentID   *int64 `json:"parent_id"`
	Content    string `json:"content"`
	LikeCount  int    `json:"like_count" gorm:"default:0"`
	ReplyCount int    `json:"reply_count" gorm:"default:0"`
	Level      int    `json:"level" gorm:"default:0"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	// Virtual fields, not stored in database
	Liked    bool         `json:"liked" gorm:"-"`
	UserInfo *UserProfile `json:"user_info,omitempty" gorm:"-"`
	Replies  []Comment    `json:"replies,omitempty" gorm:"-"`
}

// TableName specifies the table name for GORM
func (Comment) TableName() string {
	return "comments"
}

// CommentLike represents the comment_likes table structure
type CommentLike struct {
	ID        int64 `json:"id" gorm:"primaryKey"`
	CommentID int64 `json:"comment_id"`
	UserID    int64 `json:"user_id"`
	CreatedAt int64 `json:"created_at"`
}

// TableName specifies the table name for GORM
func (CommentLike) TableName() string {
	return "comment_likes"
}

// CommentDAO handles database operations for comments
type CommentDAO struct {
	db *gorm.DB
}

// NewCommentDAO creates a new CommentDAO
func NewCommentDAO(db *gorm.DB) *CommentDAO {
	return &CommentDAO{db: db}
}

// Create inserts a new comment
func (dao *CommentDAO) Create(comment *Comment) (int64, error) {
	// Set timestamps
	now := time.Now().UnixMilli()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	// Check max nesting level
	if comment.ParentID != nil {
		var parentComment Comment
		if err := tx.First(&parentComment, *comment.ParentID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, errors.New("parent comment not found")
			}
			return 0, err
		}

		// Set level based on parent
		comment.Level = parentComment.Level + 1

		// Check max nesting level (3)
		if comment.Level > 3 {
			tx.Rollback()
			return 0, errors.New("maximum comment nesting level reached")
		}

		// Increment parent's reply count
		if err := tx.Model(&Comment{}).Where("id = ?", *comment.ParentID).UpdateColumn("reply_count", gorm.Expr("reply_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// Create the comment
	if err := tx.Create(comment).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// Increment post's comment count
	if err := tx.Model(&Post{}).Where("id = ?", comment.PostID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return comment.ID, nil
}

// GetByID retrieves a comment by its ID
func (dao *CommentDAO) GetByID(id int64, currentUserID int64) (*Comment, error) {
	var comment Comment
	err := dao.db.First(&comment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment not found")
		}
		return nil, err
	}

	// Check if current user liked this comment
	var count int64
	dao.db.Model(&CommentLike{}).Where("comment_id = ? AND user_id = ?", id, currentUserID).Count(&count)
	comment.Liked = count > 0

	// Get user info
	userDAO := NewUserProfileDAO(dao.db)
	userProfile, err := userDAO.GetByID(comment.UserID)
	if err == nil {
		comment.UserInfo = userProfile
	}

	// If this is a top-level comment, get replies
	if comment.Level == 0 {
		replies, err := dao.GetReplies(id, currentUserID)
		if err != nil {
			return nil, err
		}
		comment.Replies = replies
	}

	return &comment, nil
}

// GetReplies gets replies for a comment
func (dao *CommentDAO) GetReplies(commentID int64, currentUserID int64) ([]Comment, error) {
	var replies []Comment
	err := dao.db.Where("parent_id = ?", commentID).Order("created_at ASC").Find(&replies).Error
	if err != nil {
		return nil, err
	}

	// Get additional data for each reply
	for i := range replies {
		// Check if current user liked this reply
		var count int64
		dao.db.Model(&CommentLike{}).Where("comment_id = ? AND user_id = ?", replies[i].ID, currentUserID).Count(&count)
		replies[i].Liked = count > 0

		// Get user info
		userDAO := NewUserProfileDAO(dao.db)
		userProfile, err := userDAO.GetByID(replies[i].UserID)
		if err == nil {
			replies[i].UserInfo = userProfile
		}

		// If this is not a level 3 comment, get its replies too
		if replies[i].Level < 3 {
			nestedReplies, err := dao.GetReplies(replies[i].ID, currentUserID)
			if err != nil {
				return nil, err
			}
			replies[i].Replies = nestedReplies
		}
	}

	return replies, nil
}

// ListComments retrieves comments for a post with pagination
func (dao *CommentDAO) ListComments(postID int64, page, pageSize int, currentUserID int64) ([]Comment, int64, error) {
	var comments []Comment
	var total int64

	// Only get top-level comments (level = 0)
	query := dao.db.Model(&Comment{}).Where("post_id = ? AND level = 0", postID)

	// Count total records for pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and order
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	// Get additional data for each comment
	for i := range comments {
		// Check if current user liked this comment
		var count int64
		dao.db.Model(&CommentLike{}).Where("comment_id = ? AND user_id = ?", comments[i].ID, currentUserID).Count(&count)
		comments[i].Liked = count > 0

		// Get user info
		userDAO := NewUserProfileDAO(dao.db)
		userProfile, err := userDAO.GetByID(comments[i].UserID)
		if err == nil {
			comments[i].UserInfo = userProfile
		}

		// Get replies
		replies, err := dao.GetReplies(comments[i].ID, currentUserID)
		if err != nil {
			return nil, 0, err
		}
		comments[i].Replies = replies
	}

	return comments, total, nil
}

// Delete deletes a comment and all its replies
func (dao *CommentDAO) Delete(id int64) error {
	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Get the comment to be deleted
	var comment Comment
	if err := tx.First(&comment, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return err
	}

	// Get the post ID and parent ID for later updates
	postID := comment.PostID
	parentID := comment.ParentID

	// Get reply count to subtract from post's comment count
	var replyCount int64
	if err := tx.Model(&Comment{}).Where("parent_id = ?", id).Count(&replyCount).Error; err != nil {
		tx.Rollback()
		return err
	}
	totalToSubtract := replyCount + 1 // +1 for the comment itself

	// Delete likes for this comment and its replies
	if err := dao.deleteCommentLikesRecursive(tx, id); err != nil {
		tx.Rollback()
		return err
	}

	// Delete all replies recursively
	if err := dao.deleteCommentRepliesRecursive(tx, id); err != nil {
		tx.Rollback()
		return err
	}

	// Delete the comment itself
	if err := tx.Delete(&Comment{ID: id}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update parent's reply count if this was a reply
	if parentID != nil {
		if err := tx.Model(&Comment{}).Where("id = ?", *parentID).UpdateColumn("reply_count", gorm.Expr("reply_count - ?", 1)).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update post's comment count
	if err := tx.Model(&Post{}).Where("id = ?", postID).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", totalToSubtract)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// deleteCommentLikesRecursive deletes likes for a comment and all its replies
func (dao *CommentDAO) deleteCommentLikesRecursive(tx *gorm.DB, commentID int64) error {
	// Delete likes for this comment
	if err := tx.Where("comment_id = ?", commentID).Delete(&CommentLike{}).Error; err != nil {
		return err
	}

	// Find all replies
	var replies []Comment
	if err := tx.Where("parent_id = ?", commentID).Find(&replies).Error; err != nil {
		return err
	}

	// Recursively delete likes for each reply
	for _, reply := range replies {
		if err := dao.deleteCommentLikesRecursive(tx, reply.ID); err != nil {
			return err
		}
	}

	return nil
}

// deleteCommentRepliesRecursive deletes all replies to a comment recursively
func (dao *CommentDAO) deleteCommentRepliesRecursive(tx *gorm.DB, commentID int64) error {
	// Find all replies
	var replies []Comment
	if err := tx.Where("parent_id = ?", commentID).Find(&replies).Error; err != nil {
		return err
	}

	// Recursively delete replies to each reply
	for _, reply := range replies {
		if err := dao.deleteCommentRepliesRecursive(tx, reply.ID); err != nil {
			return err
		}
	}

	// Delete all replies to this comment
	if err := tx.Where("parent_id = ?", commentID).Delete(&Comment{}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteAllForPost deletes all comments for a post
func (dao *CommentDAO) DeleteAllForPost(postID int64) error {
	// Get all comments for the post
	var comments []Comment
	if err := dao.db.Where("post_id = ?", postID).Find(&comments).Error; err != nil {
		return err
	}

	// Delete likes for all comments
	if err := dao.db.Where("comment_id IN (SELECT id FROM comments WHERE post_id = ?)", postID).Delete(&CommentLike{}).Error; err != nil {
		return err
	}

	// Delete all comments for the post
	if err := dao.db.Where("post_id = ?", postID).Delete(&Comment{}).Error; err != nil {
		return err
	}

	return nil
}

// LikeComment adds a like to a comment
func (dao *CommentDAO) LikeComment(commentID, userID int64) error {
	// Check if comment exists
	var comment Comment
	err := dao.db.First(&comment, commentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return err
	}

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Check if user already liked the comment
	var count int64
	tx.Model(&CommentLike{}).Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&count)
	if count > 0 {
		tx.Rollback()
		return errors.New("comment already liked by user")
	}

	// Add the like
	like := CommentLike{
		CommentID: commentID,
		UserID:    userID,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := tx.Create(&like).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Increment comment like count
	if err := tx.Model(&Comment{}).Where("id = ?", commentID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// UnlikeComment removes a like from a comment
func (dao *CommentDAO) UnlikeComment(commentID, userID int64) error {
	// Check if comment exists
	var comment Comment
	err := dao.db.First(&comment, commentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return err
	}

	// Start a transaction
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Check if user liked the comment
	var count int64
	tx.Model(&CommentLike{}).Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&count)
	if count == 0 {
		tx.Rollback()
		return errors.New("comment not liked by user")
	}

	// Remove the like
	if err := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).Delete(&CommentLike{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Decrement comment like count
	if err := tx.Model(&Comment{}).Where("id = ?", commentID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}
