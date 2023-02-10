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
  `bound_at` DATETIME(3) NOT NULL DEFAULT 0 COMMENT '绑定时间',
  `device_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '设备ID',
  `scheduler_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'schedulerID',
  `node_type` INT(2) NOT NULL DEFAULT 0 COMMENT '类型',
  `device_rank` INT(20) NOT NULL DEFAULT '0' COMMENT '排名',
  `device_name` CHAR(56) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '设备名称',
  `user_id` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户ID（钱包地址）',
  `sn_code` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'sn码',
  `operator` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '操作系统',
  `network_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '网络类型',
  `system_version` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版本',
  `product_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '产品类型',
  `network_info` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '网络信息',
  `external_ip` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '外网IP',
  `internal_ip` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '内网IP',
  `ip_location` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '国家地区',
  `ip_country` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '国家',
  `ip_province` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '省',
  `ip_city` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '城市',
  `latitude`FLOAT(32) NOT NULL DEFAULT '0' COMMENT '纬度',
  `longitude` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '经度',
  `mac_location` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'mac地址',
  `nat_type` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'NAT 类型',
  `upnp` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'UPNP',
  `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0'  COMMENT '丢包率',
  `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0'  COMMENT 'NAT率',
  `latency` FLOAT(32) NOT NULL DEFAULT '0'  COMMENT '线路时延',
  `cpu_usage` FLOAT(32) NOT NULL DEFAULT '0'  COMMENT 'CPU使用率',
  `cpu_cores` INT(20) NOT NULL DEFAULT '0' COMMENT 'CPU核数',
  `memory_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '内存使用率',
  `memory` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '内存',
  `disk_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '磁盘使用率',
  `disk_space` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '磁盘大小',
  `work_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '设备诊断状态',
  `device_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '设备状态',
  `bind_status` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '绑定状态',
  `disk_type` CHAR(28) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '磁盘类型',
  `active_status` INT(2) NOT NULL DEFAULT 0 COMMENT '激活状态',
  `io_system` VARCHAR(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '文件系统',
  `online_time` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '在线时长',
  `today_online_time` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日在线时长',
  `today_profit` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日收益',
  `yesterday_profit` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '昨日收益',
  `seven_days_profit` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '七天收益',
  `month_profit` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '月收益',
  `cumulative_profit` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '总收益',
  `bandwidth_up` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '上行带宽',
  `bandwidth_down` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '下行带宽',
  `total_download` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '总下行流量',
  `total_upload` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '总上行流量',
  `block_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT 'blocks',
  `download_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT '检索次数',
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
   `user_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '用户ID(钱包地址)',
   `device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
   `time` TIMESTAMP NOT NULL DEFAULT 0 COMMENT '时间',
   `income` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日收益',
   `online_time` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日在线时长',
   `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日丢包率',
   `latency` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日线路时延',
   `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日NAT率',
   `disk_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日磁盘使用率',
   `upstream_traffic` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日上行流量',
   `downstream_traffic` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '今日下行流量',
   `retrieval_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT '今日检索数量',
   `block_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT '今日blocks数量',
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
 `user_id` VARCHAR(128) NOT NULL DEFAULT ''  COMMENT '用户ID(钱包地址)',
 `device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
 `time` TIMESTAMP NOT NULL DEFAULT 0 COMMENT '时间',
 `hour_income` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '收益',
 `online_time` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '在线时长',
 `pkg_loss_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '丢包率',
 `latency` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '线路时延',
 `nat_ratio` FLOAT(32) NOT NULL DEFAULT '0' COMMENT 'NAT率',
 `disk_usage` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '磁盘使用率',
 `upstream_traffic` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '上行流量',
 `downstream_traffic` FLOAT(32) NOT NULL DEFAULT '0' COMMENT '下行流量',
 `retrieval_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT '检索数量',
 `block_count` BIGINT(20) NOT NULL DEFAULT '0' COMMENT 'blocks数量',
 PRIMARY KEY USING BTREE (`id`),
 INDEX `idx_device_info_hour_deleted_at` USING BTREE(`deleted_at` ASC)
) ENGINE = INNODB CHARSET = utf8;

DROP TABLE IF EXISTS `full_node_info`;

CREATE TABLE `full_node_info` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`total_node_count` INT(20) NOT NULL DEFAULT 0 COMMENT '全球分布式节点数量',
`validator_count` INT(20) NOT NULL DEFAULT 0 COMMENT 'L1 验证节点',
`candidate_count` INT(20) NOT NULL DEFAULT 0 COMMENT 'L1 候选节点',
`edge_count` INT(20) NOT NULL DEFAULT 0 COMMENT 'L2 边缘节点',
`total_storage` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '存储容量',
`total_upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '上行带宽',
`total_downstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '下载带宽',
`total_carfile` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '已加速Carflie',
`total_carfile_size` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '已加速Carflie大小',
`retrieval_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '加速次数',
`next_election_time` TIMESTAMP NOT NULL DEFAULT 0 COMMENT '下一轮选举时间',
`time` TIMESTAMP NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `application`;

CREATE TABLE `application` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`user_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '用户ID',
`email` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱地址',
`area_id` varchar(64) NOT NULL DEFAULT '国家地区ID',
`ip_country` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '国家',
`ip_city` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '城市',
`node_type` TINYINT(4) NOT NULL DEFAULT 0 COMMENT '节点类型',
`amount` INT(20) NOT NULL DEFAULT 0 COMMENT '数量',
`upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '上行带宽',
`disk_space` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '存储空间',
 `ip` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'IP地址',
`status` TINYINT(4) NOT NULL DEFAULT 0 COMMENT '状态',
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `application_result`;

CREATE TABLE `application_result` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`application_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '申请ID',
`user_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '用户ID',
`device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
`node_type` TINYINT(4) NOT NULL DEFAULT 0 COMMENT '节点类型',
`secret` VARCHAR(256) NOT NULL DEFAULT 0 COMMENT '密钥',
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
PRIMARY KEY (`id`)
) ENGINE = INNODB CHARSET = utf8mb4;


DROP TABLE IF EXISTS `cache_event`;

CREATE TABLE `cache_event` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
<<<<<<< Updated upstream
`device_id` VARCHAR(128) NOT NULL DEFAULT '',
`carfile_cid` VARCHAR(128) NOT NULL DEFAULT '',
`block_size` FLOAT(32) NOT NULL DEFAULT 0,
`blocks` BIGINT(20) NOT NULL DEFAULT 0,
`time` DATETIME(3) NOT NULL DEFAULT 0,
`status` TINYINT(4) NOT NULL DEFAULT 0,
=======
`device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
`carfile_cid` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'CID',
`block_size` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '文件大小',
`blocks` BIGINT(20) NOT NULL DEFAULT 0 COMMENT 'blocks',
`time` DATETIME(3) NOT NULL DEFAULT 0 COMMENT '时间',
`max_created_time` DATETIME(3) NOT NULL DEFAULT 0,
>>>>>>> Stashed changes
`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uniq_device_id_car_time` (`device_id`,`carfile_cid`,`time`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `retrieval_event`;

CREATE TABLE `retrieval_event` (
`id` BIGINT(20) NOT NULL AUTO_INCREMENT,
`device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
`blocks` BIGINT(20) NOT NULL DEFAULT 0 COMMENT 'blocks',
`time` DATETIME(3) NOT NULL DEFAULT 0 COMMENT '时间',
`upstream_bandwidth` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '上传流量',
`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ,
`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uniq_device_id_time` (`device_id`,`time`) USING BTREE
) ENGINE = INNODB CHARSET = utf8mb4;

DROP TABLE IF EXISTS `validation_event`;

CREATE TABLE `validation_event` (
 `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
 `device_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
 `validator_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '验证人',
 `blocks` BIGINT(20) NOT NULL DEFAULT 0 COMMENT 'blocks',
 `status` TINYINT(4) NOT NULL DEFAULT 0 COMMENT '状态',
 `time` DATETIME(3) NOT NULL DEFAULT 0 COMMENT '时间',
 `duration` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '完成时长',
 `upstream_traffic` FLOAT(32) NOT NULL DEFAULT 0 COMMENT '上传流量',
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