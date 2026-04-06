USE teamgram;

-- --------------------------------------------------------
--
-- 表的结构 `activities`
-- City Activity tables for city-based event app
--

CREATE TABLE `activities` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL DEFAULT 0,
  `title` VARCHAR(255) NOT NULL DEFAULT '',
  `description` TEXT,
  `photo_id` BIGINT NOT NULL DEFAULT 0,
  `city` VARCHAR(100) NOT NULL DEFAULT '',
  `start_time` BIGINT NOT NULL DEFAULT 0,
  `end_time` BIGINT NOT NULL DEFAULT 0,
  `max_participants` INT NOT NULL DEFAULT 0,
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=active 2=cancelled 3=finished',
  `is_global` TINYINT NOT NULL DEFAULT 0,
  `chat_id` BIGINT NOT NULL DEFAULT 0,
  `created_at` BIGINT NOT NULL DEFAULT 0,
  `updated_at` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_city_status` (`city`, `status`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_is_global` (`is_global`),
  FULLTEXT INDEX `idx_ft_title_desc` (`title`, `description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- 表的结构 `activity_participants`
--

CREATE TABLE `activity_participants` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `activity_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `city` VARCHAR(100) NOT NULL DEFAULT '',
  `joined_at` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_activity_user` (`activity_id`, `user_id`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- 表的结构 `activity_media`
--

CREATE TABLE `activity_media` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `activity_id` BIGINT NOT NULL,
  `photo_id` BIGINT NOT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_activity_id` (`activity_id`),
  UNIQUE INDEX `idx_activity_photo` (`activity_id`, `photo_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

