-- CDK Airdrop Station - MySQL Schema
-- 独立数据库，与社区项目隔离
CREATE DATABASE IF NOT EXISTS `cdk_airdrop` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `cdk_airdrop`;

-- 用户表 (SSO 同步)
CREATE TABLE IF NOT EXISTS `users` (
  `id`                VARCHAR(32) PRIMARY KEY,
  `username`          VARCHAR(100) NOT NULL DEFAULT '',
  `password`          VARCHAR(255) NOT NULL DEFAULT 'sso_community',
  `role`              VARCHAR(20)  NOT NULL DEFAULT 'user',
  `community_user_id` VARCHAR(100) DEFAULT '',
  `avatar`            TEXT,
  `nickname`          VARCHAR(100) DEFAULT '',
  `email`             VARCHAR(200) DEFAULT '',
  `created_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_community_uid` (`community_user_id`)
) ENGINE=InnoDB;

-- 项目表
CREATE TABLE IF NOT EXISTS `projects` (
  `id`          VARCHAR(32) PRIMARY KEY,
  `name`        VARCHAR(200) NOT NULL,
  `description` TEXT,
  `status`      VARCHAR(20)  NOT NULL DEFAULT 'active',
  `creator_id`  VARCHAR(32)  DEFAULT '',
  `created_at`  VARCHAR(30)  NOT NULL,
  `updated_at`  VARCHAR(30)  NOT NULL,
  INDEX `idx_creator` (`creator_id`),
  INDEX `idx_status`  (`status`)
) ENGINE=InnoDB;

-- 活动表 (每个项目可有多个活动/CDK池)
CREATE TABLE IF NOT EXISTS `campaigns` (
  `id`                      VARCHAR(32) PRIMARY KEY,
  `project_id`              VARCHAR(32)  NOT NULL,
  `name`                    VARCHAR(200) NOT NULL,
  `description`             TEXT,
  `status`                  VARCHAR(20)  NOT NULL DEFAULT 'active',
  `total_stock`             INT          NOT NULL DEFAULT 0,
  `claimed_count`           INT          NOT NULL DEFAULT 0,
  `remaining_count`         INT          NOT NULL DEFAULT 0,
  `start_at`                VARCHAR(30)  DEFAULT '',
  `end_at`                  VARCHAR(30)  DEFAULT '',
  `allow_repeat`            TINYINT(1)   NOT NULL DEFAULT 0,
  `per_user_limit`          INT          NOT NULL DEFAULT 1,
  `per_ip_limit`            INT          NOT NULL DEFAULT 0,
  `per_device_limit`        INT          NOT NULL DEFAULT 0,
  `require_captcha_default` TINYINT(1)   NOT NULL DEFAULT 1,
  `project_code`            VARCHAR(20)  DEFAULT '',
  `enabled`                 TINYINT(1)   NOT NULL DEFAULT 1,
  `rules`                   TEXT,
  `created_at`              VARCHAR(30)  NOT NULL,
  `updated_at`              VARCHAR(30)  NOT NULL,
  INDEX `idx_project` (`project_id`),
  INDEX `idx_status`  (`status`)
) ENGINE=InnoDB;

-- CDK 码表
CREATE TABLE IF NOT EXISTS `cdks` (
  `id`                   VARCHAR(32)  PRIMARY KEY,
  `campaign_id`          VARCHAR(32)  NOT NULL,
  `code`                 VARCHAR(500) NOT NULL,
  `status`               VARCHAR(20)  NOT NULL DEFAULT 'unused',
  `claimed_by_record_id` VARCHAR(32)  DEFAULT '',
  `claimed_at`           VARCHAR(30)  DEFAULT '',
  `node_id`              VARCHAR(32)  DEFAULT '',
  `created_at`           VARCHAR(30)  NOT NULL,
  `updated_at`           VARCHAR(30)  NOT NULL,
  INDEX `idx_campaign_status` (`campaign_id`, `status`),
  INDEX `idx_code` (`code`(100))
) ENGINE=InnoDB;

-- 分发节点表
CREATE TABLE IF NOT EXISTS `nodes` (
  `id`                   VARCHAR(32)  PRIMARY KEY,
  `project_id`           VARCHAR(32)  NOT NULL DEFAULT '',
  `campaign_id`          VARCHAR(32)  NOT NULL,
  `name`                 VARCHAR(200) NOT NULL,
  `slug`                 VARCHAR(100) NOT NULL,
  `status`               VARCHAR(20)  NOT NULL DEFAULT 'active',
  `title`                VARCHAR(200) DEFAULT '',
  `description`          TEXT,
  `button_text`          VARCHAR(100) DEFAULT '立即领取',
  `require_captcha`      TINYINT(1)   NOT NULL DEFAULT 1,
  `show_stock`           TINYINT(1)   NOT NULL DEFAULT 1,
  `show_end_time`        TINYINT(1)   NOT NULL DEFAULT 1,
  `visits`               INT          NOT NULL DEFAULT 0,
  `unique_visitors`      INT          NOT NULL DEFAULT 0,
  `claims`               INT          NOT NULL DEFAULT 0,
  `failed_claims`        INT          NOT NULL DEFAULT 0,
  `last_visited_at`      VARCHAR(30)  DEFAULT '',
  `limit_val`            INT          NOT NULL DEFAULT 0,
  `ip_limit_enabled`     TINYINT(1)   NOT NULL DEFAULT 0,
  `device_limit_enabled` TINYINT(1)   NOT NULL DEFAULT 0,
  `created_at`           VARCHAR(30)  NOT NULL,
  `updated_at`           VARCHAR(30)  NOT NULL,
  UNIQUE KEY `uk_slug` (`slug`),
  INDEX `idx_project`  (`project_id`),
  INDEX `idx_campaign` (`campaign_id`)
) ENGINE=InnoDB;

-- 领取记录表
CREATE TABLE IF NOT EXISTS `claim_records` (
  `id`              VARCHAR(32)  PRIMARY KEY,
  `campaign_id`     VARCHAR(32)  NOT NULL DEFAULT '',
  `node_id`         VARCHAR(32)  NOT NULL DEFAULT '',
  `project_id`      VARCHAR(32)  NOT NULL DEFAULT '',
  `cdk_id`          VARCHAR(32)  DEFAULT '',
  `code`            VARCHAR(500) DEFAULT '',
  `status`          VARCHAR(20)  NOT NULL,
  `reason`          TEXT,
  `ip`              VARCHAR(50)  DEFAULT '',
  `user_agent`      TEXT,
  `fingerprint`     VARCHAR(200) DEFAULT '',
  `idempotency_key` VARCHAR(200) DEFAULT '',
  `claim_token`     VARCHAR(100) DEFAULT '',
  `hcaptcha_passed` TINYINT(1)   NOT NULL DEFAULT 0,
  `risk_hit`        TINYINT(1)   NOT NULL DEFAULT 0,
  `risk_rule_ids`   TEXT,
  `user_id`         VARCHAR(32)  DEFAULT '',
  `reward_content`  TEXT,
  `created_at`      VARCHAR(30)  NOT NULL,
  INDEX `idx_campaign`    (`campaign_id`),
  INDEX `idx_node`        (`node_id`),
  INDEX `idx_project`     (`project_id`),
  INDEX `idx_status`      (`status`),
  INDEX `idx_fingerprint` (`fingerprint`),
  INDEX `idx_ip`          (`ip`),
  INDEX `idx_created`     (`created_at`)
) ENGINE=InnoDB;

-- 风控规则表
CREATE TABLE IF NOT EXISTS `risk_rules` (
  `id`         VARCHAR(32)  PRIMARY KEY,
  `name`       VARCHAR(200) NOT NULL,
  `type`       VARCHAR(50)  NOT NULL,
  `enabled`    TINYINT(1)   NOT NULL DEFAULT 1,
  `config`     JSON,
  `action`     VARCHAR(50)  DEFAULT 'block',
  `created_at` VARCHAR(30)  NOT NULL,
  `updated_at` VARCHAR(30)  NOT NULL
) ENGINE=InnoDB;

-- 黑名单表
CREATE TABLE IF NOT EXISTS `blacklist` (
  `id`         VARCHAR(32)  PRIMARY KEY,
  `type`       VARCHAR(50)  NOT NULL,
  `value`      VARCHAR(500) NOT NULL,
  `reason`     TEXT,
  `enabled`    TINYINT(1)   NOT NULL DEFAULT 1,
  `created_at` VARCHAR(30)  NOT NULL,
  `updated_at` VARCHAR(30)  NOT NULL,
  INDEX `idx_type_value` (`type`, `value`(100))
) ENGINE=InnoDB;

-- 系统日志表
CREATE TABLE IF NOT EXISTS `system_logs` (
  `id`          VARCHAR(32)  PRIMARY KEY,
  `type`        VARCHAR(30)  NOT NULL DEFAULT 'operation',
  `level`       VARCHAR(10)  NOT NULL DEFAULT 'info',
  `title`       VARCHAR(200) DEFAULT '',
  `message`     TEXT,
  `actor`       VARCHAR(100) DEFAULT '',
  `ip`          VARCHAR(50)  DEFAULT '',
  `target_type` VARCHAR(50)  DEFAULT '',
  `target_id`   VARCHAR(50)  DEFAULT '',
  `metadata`    JSON,
  `created_at`  VARCHAR(30)  NOT NULL,
  INDEX `idx_type`    (`type`),
  INDEX `idx_level`   (`level`),
  INDEX `idx_created` (`created_at`)
) ENGINE=InnoDB;

-- 导出任务表
CREATE TABLE IF NOT EXISTS `export_tasks` (
  `id`          VARCHAR(32)  PRIMARY KEY,
  `type`        VARCHAR(50)  NOT NULL,
  `status`      VARCHAR(20)  NOT NULL DEFAULT 'pending',
  `filename`    VARCHAR(200) DEFAULT '',
  `file_path`   VARCHAR(500) DEFAULT '',
  `filter`      JSON,
  `error`       TEXT,
  `created_at`  VARCHAR(30)  NOT NULL,
  `finished_at` VARCHAR(30)  DEFAULT ''
) ENGINE=InnoDB;

-- 系统设置 (单行)
CREATE TABLE IF NOT EXISTS `settings` (
  `id`                INT          PRIMARY KEY DEFAULT 1,
  `system_name`       VARCHAR(100) DEFAULT 'Zens-CDK',
  `brand_name`        VARCHAR(100) DEFAULT 'Zens-CDK',
  `brand_english_name` VARCHAR(100) DEFAULT 'Zens CDK Airdrop Hub',
  `logo_text`         VARCHAR(20)  DEFAULT 'ZC',
  `public_base_url`   VARCHAR(500) DEFAULT '',
  `storage_mode`      VARCHAR(20)  DEFAULT 'mysql',
  `redis_enabled`     TINYINT(1)   DEFAULT 0,
  `rabbitmq_enabled`  TINYINT(1)   DEFAULT 0,
  `created_at`        VARCHAR(30)  NOT NULL,
  `updated_at`        VARCHAR(30)  NOT NULL
) ENGINE=InnoDB;

-- 验证码配置 (单行)
CREATE TABLE IF NOT EXISTS `captcha_config` (
  `id`                  INT         PRIMARY KEY DEFAULT 1,
  `provider`            VARCHAR(20) DEFAULT 'hcaptcha',
  `enabled`             TINYINT(1)  DEFAULT 1,
  `last_test_status`    VARCHAR(20) DEFAULT '',
  `last_test_message`   TEXT,
  `last_test_at`        VARCHAR(30) DEFAULT ''
) ENGINE=InnoDB;

-- 初始化默认配置行
INSERT IGNORE INTO `settings` (`id`, `created_at`, `updated_at`) VALUES (1, NOW(), NOW());
INSERT IGNORE INTO `captcha_config` (`id`) VALUES (1);
