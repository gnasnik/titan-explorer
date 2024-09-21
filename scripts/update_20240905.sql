DROP TABLE IF EXISTS `storage_stats`;
DROP TABLE IF EXISTS `projects`;
DROP TABLE IF EXISTS `system_info`;


ALTER TABLE assets ADD COLUMN backup_result int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN edge_replicas int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN candidate_replicas int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN total_blocks int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN created_time timestamp NOT NULL DEFAULT '0000-00-00';
ALTER TABLE assets ADD COLUMN area_id VARCHAR(256) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN state VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN note VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN bandwidth int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN source int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN retry_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN replenish_replicas int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN failed_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN succeeded_count int(20) NOT NULL DEFAULT 0;

ALTER TABLE assets DROP COLUMN `project_id`;
ALTER TABLE assets DROP COLUMN `event`;
ALTER TABLE assets DROP COLUMN `type`;
ALTER TABLE assets DROP COLUMN `name`;
ALTER TABLE assets DROP COLUMN `created_at`;
ALTER TABLE assets DROP COLUMN `updated_at`;
ALTER TABLE assets DROP COLUMN `deleted_at`;

--  full node info
ALTER TABLE full_node_info DROP COLUMN `validator_count`;
ALTER TABLE full_node_info DROP COLUMN `online_validator_count`;
ALTER TABLE full_node_info DROP COLUMN `next_election_time`;
ALTER TABLE full_node_info DROP COLUMN `fvm_order_count`;
ALTER TABLE full_node_info DROP COLUMN `f_high`;
ALTER TABLE full_node_info DROP COLUMN `t_next_election_high`;


-- device_info
ALTER TABLE device_info ADD COLUMN asset_succeeded_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN asset_failed_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN retrieve_succeeded_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN retrieve_failed_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN project_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN project_succeeded_count int(20) NOT NULL DEFAULT 0;
ALTER TABLE device_info ADD COLUMN project_failed_count int(20) NOT NULL DEFAULT 0;


DROP TABLE IF EXISTS `area_config`;
CREATE TABLE `area_config` (
`area_id` varchar(255) NOT NULL DEFAULT '',
`name_cn` varchar(255) NOT NULL DEFAULT '',
`name_en` varchar(255) NOT NULL DEFAULT '',
PRIMARY KEY (`area_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;