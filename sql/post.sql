-- Post table
CREATE TABLE `posts` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL COMMENT 'User who created the post',
  `content` text DEFAULT NULL COMMENT 'Post text content',
  `view_count` int(11) NOT NULL DEFAULT 0 COMMENT 'Number of views',
  `like_count` int(11) NOT NULL DEFAULT 0 COMMENT 'Number of likes',
  `comment_count` int(11) NOT NULL DEFAULT 0 COMMENT 'Number of comments',
  `created_at` bigint(20) NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
  `updated_at` bigint(20) NOT NULL COMMENT 'Last update timestamp in milliseconds since epoch',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Post Image table
CREATE TABLE `post_images` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `post_id` bigint(20) NOT NULL COMMENT 'Post this image belongs to',
  `image_path` varchar(255) NOT NULL COMMENT 'Path to the image file',
  `display_order` int(11) NOT NULL DEFAULT 0 COMMENT 'Order for displaying multiple images',
  `created_at` bigint(20) NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
  PRIMARY KEY (`id`),
  KEY `idx_post_id` (`post_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Comment table
CREATE TABLE `comments` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `post_id` bigint(20) NOT NULL COMMENT 'Post this comment belongs to',
  `user_id` bigint(20) NOT NULL COMMENT 'User who created the comment',
  `parent_id` bigint(20) DEFAULT NULL COMMENT 'Parent comment ID for nested comments',
  `content` text NOT NULL COMMENT 'Comment text content',
  `like_count` int(11) NOT NULL DEFAULT 0 COMMENT 'Number of likes',
  `reply_count` int(11) NOT NULL DEFAULT 0 COMMENT 'Number of replies',
  `level` int(11) NOT NULL DEFAULT 0 COMMENT 'Nesting level (0 for top-level comments)',
  `created_at` bigint(20) NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
  `updated_at` bigint(20) NOT NULL COMMENT 'Last update timestamp in milliseconds since epoch',
  PRIMARY KEY (`id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Like table for posts
CREATE TABLE `post_likes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `post_id` bigint(20) NOT NULL COMMENT 'Post that was liked',
  `user_id` bigint(20) NOT NULL COMMENT 'User who liked the post',
  `created_at` bigint(20) NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_post_user` (`post_id`,`user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Like table for comments
CREATE TABLE `comment_likes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `comment_id` bigint(20) NOT NULL COMMENT 'Comment that was liked',
  `user_id` bigint(20) NOT NULL COMMENT 'User who liked the comment',
  `created_at` bigint(20) NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_comment_user` (`comment_id`,`user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
