CREATE DATABASE IF NOT EXISTS teamgram_stickers;
USE teamgram_stickers;

CREATE TABLE IF NOT EXISTS sticker_sets (
  id              BIGINT NOT NULL AUTO_INCREMENT,
  set_id          BIGINT NOT NULL,
  access_hash     BIGINT NOT NULL DEFAULT 0,
  short_name      VARCHAR(128) NOT NULL,
  title           VARCHAR(256) NOT NULL DEFAULT '',
  sticker_type    VARCHAR(32) NOT NULL DEFAULT 'regular',
  is_animated     TINYINT(1) NOT NULL DEFAULT 0,
  is_video        TINYINT(1) NOT NULL DEFAULT 0,
  is_masks        TINYINT(1) NOT NULL DEFAULT 0,
  is_emojis       TINYINT(1) NOT NULL DEFAULT 0,
  is_official     TINYINT(1) NOT NULL DEFAULT 0,
  sticker_count   INT NOT NULL DEFAULT 0,
  hash            INT NOT NULL DEFAULT 0,
  thumb_doc_id    BIGINT NOT NULL DEFAULT 0,
  data_json       MEDIUMTEXT NOT NULL,
  fetched_at      BIGINT NOT NULL DEFAULT 0,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_short_name (short_name),
  UNIQUE KEY idx_set_id (set_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sticker_set_documents (
  id                 BIGINT NOT NULL AUTO_INCREMENT,
  set_id             BIGINT NOT NULL,
  document_id        BIGINT NOT NULL,
  sticker_index      INT NOT NULL DEFAULT 0,
  emoji              VARCHAR(64) NOT NULL DEFAULT '',
  bot_file_id        VARCHAR(512) NOT NULL DEFAULT '',
  bot_file_unique_id VARCHAR(256) NOT NULL DEFAULT '',
  bot_thumb_file_id  VARCHAR(512) NOT NULL DEFAULT '',
  document_data      MEDIUMTEXT NOT NULL,
  file_downloaded    TINYINT(1) NOT NULL DEFAULT 0,
  created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_set_id (set_id),
  UNIQUE KEY idx_document_id (document_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_recent_stickers (
  id            BIGINT NOT NULL AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  document_id   BIGINT NOT NULL,
  emoji         VARCHAR(64) NOT NULL DEFAULT '',
  document_data MEDIUMTEXT NOT NULL,
  date2         BIGINT NOT NULL DEFAULT 0,
  deleted       TINYINT(1) NOT NULL DEFAULT 0,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_user_document (user_id, document_id),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_faved_stickers (
  id            BIGINT NOT NULL AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  document_id   BIGINT NOT NULL,
  emoji         VARCHAR(64) NOT NULL DEFAULT '',
  document_data MEDIUMTEXT NOT NULL,
  date2         BIGINT NOT NULL DEFAULT 0,
  deleted       TINYINT(1) NOT NULL DEFAULT 0,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_user_document (user_id, document_id),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
