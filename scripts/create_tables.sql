DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`uuid` VARCHAR(255) NOT NULL DEFAULT '',
`username` VARCHAR(255) NOT NULL DEFAULT '',
`pass_hash` VARCHAR(255) NOT NULL DEFAULT '',
`user_email` VARCHAR(255) NOT NULL DEFAULT '',
`address` VARCHAR(255) NOT NULL DEFAULT '',
`role` TINYINT(4) NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
`deleted_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `login_log`;

CREATE TABLE `login_log` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`login_username` VARCHAR(50) NOT NULL DEFAULT '',
`ip_address` VARCHAR(50) NOT NULL DEFAULT '',
`login_location` VARCHAR(255) NOT NULL DEFAULT '',
`browser` VARCHAR(50) NOT NULL DEFAULT '',
`os` VARCHAR(50) NOT NULL DEFAULT '',
`status` TINYINT(4) NOT NULL DEFAULT 0,
`msg` VARCHAR(255) NOT NULL DEFAULT '',
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY USING BTREE (`id`)
) ENGINE = INNODB CHARACTER SET = utf8mb4;

DROP TABLE IF EXISTS `operation_log`;

CREATE TABLE `operation_log` (
`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
`title` VARCHAR(50) NOT NULL DEFAULT '',
`business_type` INT(2) NOT NULL DEFAULT 0,
`method` VARCHAR(100) NOT NULL DEFAULT '',
`request_method` VARCHAR(10) NOT NULL DEFAULT '',
`operator_type` INT(1) NOT NULL DEFAULT 0,
`operator_username` VARCHAR(50) NOT NULL DEFAULT '',
`operator_url` VARCHAR(500) NOT NULL DEFAULT '',
`operator_ip` VARCHAR(50) NOT NULL DEFAULT '',
`operator_location` VARCHAR(255) NOT NULL DEFAULT '',
`operator_param` VARCHAR(2000) NOT NULL DEFAULT '',
`json_result` VARCHAR(2000) NOT NULL DEFAULT '',
`status` INT(1) NOT NULL DEFAULT 0,
`error_msg` VARCHAR(2000) NOT NULL DEFAULT '',
`created_at` DATETIME(6) NOT NULL DEFAULT 0,
`updated_at` DATETIME(6) NOT NULL DEFAULT 0,
PRIMARY KEY USING BTREE (`id`)
) ENGINE = INNODB CHARACTER SET = utf8mb4;

DROP TABLE IF EXISTS `schedulers`;

CREATE TABLE `schedulers` (
 `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
 `uuid` VARCHAR(255) NOT NULL DEFAULT '',
 `area` VARCHAR(255) NOT NULL DEFAULT '',
 `address` VARCHAR(255) NOT NULL DEFAULT '',
 `status` INT(1) NOT NULL DEFAULT 0,
 `token` VARCHAR(255) NOT NULL DEFAULT '',
 `created_at` DATETIME(3) NOT NULL DEFAULT 0,
 `updated_at` DATETIME(3) NOT NULL DEFAULT 0,
 `deleted_at` DATETIME(3) NOT NULL DEFAULT 0,
 PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `device_info`;

CREATE TABLE `device_info` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME(3) NOT NULL DEFAULT 0,
  `updated_at` DATETIME(3) NOT NULL DEFAULT 0,
  `deleted_at` DATETIME(3) NOT NULL DEFAULT 0,
  `bound_at` DATETIME(3) NOT NULL DEFAULT 0,
  `device_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `scheduler_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `node_type` INT(2) NOT NULL DEFAULT 0,
  `device_rank` INT(20) NOT NULL DEFAULT '0' COMMENT '',
  `device_name` CHAR(56) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `user_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `sn_code` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `operator` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `network_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `system_version` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `product_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `network_info` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `external_ip` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `internal_ip` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `ip_location` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `ip_country` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `ip_province` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `ip_city` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `latitude`FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `longitude` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `mac_location` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `nat_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `upnp` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT 'Nat',
  `latency` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `cpu_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `cpu_cores` INT(20) NOT NULL DEFAULT '0' COMMENT '',
  `memory_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `memory` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `disk_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `disk_space` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '',
  `bind_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `work_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `active_status` INT(2) NOT NULL DEFAULT 0,
  `disk_type` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `io_system` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `online_time` FLOAT(32) NOT NULL DEFAULT '0',
  `today_online_time` FLOAT(32) NOT NULL DEFAULT '0',
  `today_profit` FLOAT(32) NOT NULL DEFAULT '0',
  `yesterday_profit` FLOAT(32) NOT NULL DEFAULT '0',
  `seven_days_profit` FLOAT(32) NOT NULL DEFAULT '0',
  `month_profit` FLOAT(32) NOT NULL DEFAULT '0',
  `cumulative_profit` FLOAT(32) NOT NULL DEFAULT '0',
  `bandwidth_up` FLOAT(32) NOT NULL DEFAULT '0',
  `bandwidth_down` FLOAT(32) NOT NULL DEFAULT '0',
  `total_download` FLOAT(32) NOT NULL DEFAULT '0',
  `total_upload` FLOAT(32) NOT NULL DEFAULT '0',
  `block_count` BIGINT(20) NOT NULL DEFAULT '0',
  `download_count` BIGINT(20) NOT NULL DEFAULT '0',
  PRIMARY KEY USING BTREE (`id`),
  UNIQUE KEY `idx_device_id` (`device_id`) USING BTREE,
  INDEX `idx_device_info_deleted_at` USING BTREE(`deleted_at` ASC)
) ENGINE = INNODB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4;

DROP TABLE IF EXISTS `device_info_daily`;

CREATE TABLE `device_info_daily` (
   `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
   `created_at` TIMESTAMP NOT NULL DEFAULT 0,
   `updated_at` TIMESTAMP NOT NULL DEFAULT 0,
   `deleted_at` DATETIME(3) NOT NULL DEFAULT 0,
   `user_id` VARCHAR(128) NOT NULL DEFAULT '',
   `device_id` VARCHAR(128) NOT NULL DEFAULT '',
   `time` TIMESTAMP NOT NULL DEFAULT 0,
   `income` FLOAT(32) NOT NULL DEFAULT '0',
   `online_time` FLOAT(32) NOT NULL DEFAULT '0',
   `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0',
   `latency` FLOAT(32) NOT NULL DEFAULT '0',
   `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0',
   `disk_usage` FLOAT(32) NOT NULL DEFAULT '0',
   `upstream_traffic` FLOAT(32) NOT NULL DEFAULT '0',
   `downstream_traffic` FLOAT(32) NOT NULL DEFAULT '0',
   `retrieval_count` BIGINT(20) NOT NULL DEFAULT '0',
   `block_count` BIGINT(20) NOT NULL DEFAULT '0',
   PRIMARY KEY USING BTREE (`id`),
   UNIQUE KEY `idx_device_id_time` (`device_id`,`time`) USING BTREE,
   INDEX `idx_device_info_daily_deleted_at` USING BTREE(`deleted_at` ASC)
) ENGINE = INNODB CHARSET = utf8;

DROP TABLE IF EXISTS `device_info_hour`;

CREATE TABLE `device_info_hour` (
 `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
 `created_at` TIMESTAMP NOT NULL DEFAULT 0,
 `updated_at` TIMESTAMP NOT NULL DEFAULT 0,
 `deleted_at` DATETIME(3) NOT NULL DEFAULT 0,
 `user_id` VARCHAR(128) NOT NULL DEFAULT '',
 `device_id` VARCHAR(128) NOT NULL DEFAULT '',
 `time` TIMESTAMP NOT NULL DEFAULT 0,
 `hour_income` FLOAT(32) NOT NULL DEFAULT '0',
 `online_time` FLOAT(32) NOT NULL DEFAULT '0',
 `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0',
 `latency` FLOAT(32) NOT NULL DEFAULT '0',
 `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0',
 `disk_usage` FLOAT(32) NOT NULL DEFAULT '0',
 `upstream_traffic` FLOAT(32) NOT NULL DEFAULT '0',
 `downstream_traffic` FLOAT(32) NOT NULL DEFAULT '0',
 `retrieval_count` BIGINT(20) NOT NULL DEFAULT '0',
 `block_count` BIGINT(20) NOT NULL DEFAULT '0',
 PRIMARY KEY USING BTREE (`id`),
 INDEX `idx_device_info_hour_deleted_at` USING BTREE(`deleted_at` ASC)
) ENGINE = INNODB CHARSET = utf8;

DROP TABLE IF EXISTS `full_node_info`;

CREATE TABLE `full_node_info` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`total_node_count` INT(20) NOT NULL DEFAULT 0,
`validator_count` INT(20) NOT NULL DEFAULT 0,
`candidate_count` INT(20) NOT NULL DEFAULT 0,
`edge_count` INT(20) NOT NULL DEFAULT 0,
`total_storage` FLOAT(32) NOT NULL DEFAULT 0,
`total_upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0,
`total_downstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0,
`total_carfile` BIGINT(20) NOT NULL DEFAULT 0,
`total_carfile_size` FLOAT(32) NOT NULL DEFAULT 0,
`retrieval_count` BIGINT(20) NOT NULL DEFAULT 0,
`next_election_time` TIMESTAMP NOT NULL DEFAULT 0,
`time` TIMESTAMP NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `application`;

CREATE TABLE `application` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`user_id` VARCHAR(128) NOT NULL DEFAULT '',
`email` VARCHAR(128) NOT NULL DEFAULT '',
`area_id` varchar(64) NOT NULL DEFAULT '',
`ip_country` VARCHAR(128) NOT NULL DEFAULT '',
`ip_city` VARCHAR(128) NOT NULL DEFAULT '',
`node_type` TINYINT(4) NOT NULL DEFAULT 0,
`amount` INT(20) NOT NULL DEFAULT 0,
`upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0,
`disk_space` FLOAT(32) NOT NULL DEFAULT 0,
 `ip` VARCHAR(128) NOT NULL DEFAULT '',
`status` TINYINT(4) NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `application_result`;

CREATE TABLE `application_result` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`application_id` BIGINT(20) NOT NULL DEFAULT 0,
`user_id` VARCHAR(128) NOT NULL DEFAULT '',
`device_id` VARCHAR(128) NOT NULL DEFAULT '',
`node_type` TINYINT(4) NOT NULL DEFAULT 0,
`secret` VARCHAR(256) NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `cache_event`;

CREATE TABLE `cache_event` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`device_id` VARCHAR(128) NOT NULL DEFAULT '',
`carfile_cid` VARCHAR(128) NOT NULL DEFAULT '',
`block_size` FLOAT(32) NOT NULL DEFAULT 0,
`blocks` BIGINT(20) NOT NULL DEFAULT 0,
`time` DATETIME(3) NOT NULL DEFAULT 0,
`status` TINYINT(4) NOT NULL DEFAULT 0,
`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uniq_device_id_car_time` (`device_id`,`carfile_cid`,`time`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `retrieval_event`;

CREATE TABLE `retrieval_event` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`device_id` VARCHAR(128) NOT NULL DEFAULT '',
`blocks` BIGINT(20) NOT NULL DEFAULT 0,
`time` DATETIME(3) NOT NULL DEFAULT 0,
`upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0,
`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uniq_device_id_time` (`device_id`,`time`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `validation_event`;

CREATE TABLE `validation_event` (
 `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
 `device_id` VARCHAR(128) NOT NULL DEFAULT '',
 `validator_id` VARCHAR(128) NOT NULL DEFAULT '',
 `blocks` BIGINT(20) NOT NULL DEFAULT 0,
 `status` TINYINT(4) NOT NULL DEFAULT 0,
 `time` DATETIME(3) NOT NULL DEFAULT 0,
 `duration` BIGINT(20) NOT NULL DEFAULT 0,
 `upstream_traffic` FLOAT(32) NOT NULL DEFAULT 0,
 `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY (`id`),
 UNIQUE KEY `uniq_device_id_time` (`device_id`,`time`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `system_info`;

CREATE TABLE `system_info` (
 `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
 `scheduler_uuid` VARCHAR(128) NOT NULL DEFAULT '',
 `car_file_count` BIGINT(20) NOT NULL DEFAULT 0,
 `download_count` BIGINT(20) NOT NULL DEFAULT 0,
 `next_election_time` BIGINT(20) NOT NULL DEFAULT 0,
 `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY (`id`),
 UNIQUE KEY `uniq_uuid` (`scheduler_uuid`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;