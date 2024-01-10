ALTER TABLE fil_storage ADD COLUMN gas FLOAT(32) NOT NULL DEFAULT 0;
ALTER TABLE fil_storage ADD COLUMN pledge FLOAT(32) NOT NULL DEFAULT 0;


ALTER TABLE assets ADD COLUMN user_id VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN project_id BIGINT(20) NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN project_id BIGINT(20) NOT NULL DEFAULT 0;


ALTER TABLE storage_stats ADD COLUMN locations VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE storage_stats ADD COLUMN s_rank BIGINT(20) NOT NULL DEFAULT 0;

ALTER  TABLE  device_info ADD COLUMN device_status_code BIGINT(20) NOT NULL DEFAULT 0 AFTER device_status;

-- 2024-01-01
ALTER TABLE users ADD COLUMN referral_code VARCHAR(64) NOT NULL DEFAULT '' AFTER project_id;
ALTER TABLE users ADD COLUMN referrer VARCHAR(64) NOT NULL DEFAULT '' AFTER referral_code;

ALTER TABLE users ADD COLUMN reward  BIGINT(20) NOT NULL DEFAULT 0 AFTER referrer;
ALTER TABLE users ADD COLUMN payout  BIGINT(20) NOT NULL DEFAULT 0 AFTER reward;

ALTER TABLE users RENAME COLUMN address TO wallet_address;