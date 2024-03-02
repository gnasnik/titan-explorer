create table sign_info(
`miner_id` varchar(255),
`address` varchar(255),
`date` bigint,
`signed_msg` varchar(1024),
primary key(miner_id)
);

create table epoch_rewards (
`id` bigint(20) NOT NULL AUTO_INCREMENT,
`username`  VARCHAR(128) NOT NULL DEFAULT '',
`referral_earned` bigint(20) NOT NULL DEFAULT 0,
`points_earned` bigint(20) NOT NULL DEFAULT 0,
`total_uptime` bigint(20) NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
)

create table epoch_info(
`id` bigint(20) NOT NULL AUTO_INCREMENT,
`name`  VARCHAR(128) NOT NULL DEFAULT '',
`point_name` VARCHAR(128) NOT NULL DEFAULT '',
`start_date` DATETIME(3) NOT NULL DEFAULT 0,
`end_date`  DATETIME(3) NOT NULL DEFAULT 0,
`created_at` DATETIME(3) NOT NULL DEFAULT 0,
`updated_at` DATETIME(3) NOT NULL DEFAULT 0,
)


ALTER  TABLE  device_info ADD COLUMN nat_type VARCHAR(128) NOT NULL DEFAULT '' AFTER retrieval_count;

ALTER TABLE full_node_info ADD COLUMN  online_validator_count bigint(20) NOT NULL DEFAULT 0 AFTER validator_count;
ALTER TABLE full_node_info ADD COLUMN  online_candidate_count bigint(20) NOT NULL DEFAULT 0 AFTER candidate_count;
ALTER TABLE full_node_info ADD COLUMN  online_edge_count bigint(20) NOT NULL DEFAULT 0 AFTER edge_count;