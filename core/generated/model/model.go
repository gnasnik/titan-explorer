package model

import "time"

type Language string

const (
	LanguageEN = "en"
	LanguageCN = "cn"
)

type UserRole int

const (
	UserRoleDefault UserRole = iota
	UserRoleAdmin
	UserRoleKOL
)

var SupportLanguages = []Language{LanguageEN, LanguageCN}

type RewardEvent string

var (
	RewardEventInviteFrens RewardEvent = "invite_frens"
	RewardEventBindDevice  RewardEvent = "bind_device"
	RewardEventEarning     RewardEvent = "earning"
	RewardEventReferrals   RewardEvent = "referrals"
)

type Project struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"-"`
	UpdatedAt time.Time `db:"updated_at" json:"-"`
}

type StorageStats struct {
	ID                         int64     `db:"id" json:"id"`
	Rank                       int64     `db:"s_rank" json:"s_rank"`
	ProjectId                  int64     `db:"project_id" json:"project_id"`
	ProjectName                string    `db:"project_name" json:"project_name"`
	TotalSize                  int64     `db:"total_size" json:"total_size"`
	UserCount                  int64     `db:"user_count" json:"user_count"`
	ProviderCount              int64     `db:"provider_count" json:"provider_count"`
	Expiration                 time.Time `db:"expiration" json:"expiration"`
	Time                       string    `db:"time" json:"time"`
	StorageChange24H           int64     `db:"storage_change_24h" json:"storage_change_24h"`
	StorageChangePercentage24H float64   `db:"storage_change_percentage_24h" json:"storage_change_percentage_24h"`
	Gas                        float64   `db:"gas" json:"gas"`
	Pledge                     float64   `db:"pledge" json:"pledge"`
	Locations                  string    `db:"locations" json:"locations"`
	CreatedAt                  time.Time `db:"created_at" json:"-"`
	UpdatedAt                  time.Time `db:"updated_at" json:"-"`
}

type StorageSummary struct {
	TotalSize         float64 `db:"total_size" json:"total_size"`
	Projects          int64   `db:"projects" json:"projects"`
	Users             int64   `db:"users" json:"users"`
	Pledges           float64 `db:"pledges" json:"pledges"`
	Gases             float64 `db:"gases" json:"gases"`
	Providers         int64   `db:"providers" json:"providers"`
	RetrievalProvider int64   `db:"retrieval_providers" json:"retrieval_providers"`
	StorageProvider   int64   `db:"storage_providers" json:"storage_providers"`
	LatestUpdateTime  string  `db:"-" json:"latest_update_time"`
}

type StorageProvider struct {
	ID          int64     `db:"id" json:"id"`
	ProviderID  string    `db:"provider_id" json:"provider_id"`
	IP          string    `db:"ip" json:"ip"`
	Location    string    `db:"location" json:"location"`
	Retrievable bool      `db:"retrievable" json:"retrievable"`
	CreatedAt   time.Time `db:"created_at" json:"-"`
	UpdatedAt   time.Time `db:"updated_at" json:"-"`
}

type InviteFrensRecord struct {
	Email      string    `db:"email" json:"email"`
	Status     int       `db:"status" json:"status"`
	BoundCount int       `db:"bound_count" json:"bound_count"`
	Reward     float64   `db:"reward" json:"reward"`
	Referrer   string    `db:"referrer" json:"referrer"`
	Time       time.Time `db:"time" json:"time"`
}

type SignInfo struct {
	MinerID      string `json:"miner_id" db:"miner_id"`
	Address      string `json:"address" db:"address"`
	Date         int64  `json:"date" db:"date"`
	SignedMsg    string `json:"signed_msg" db:"signed_msg"`
	MinerPower   string `json:"miner_power" db:"miner_power"`
	MinerBalance string `json:"miner_balance" db:"miner_balance"`
}

type DeviceDistribution struct {
	Country string `json:"country" db:"country"`
	Count   int    `json:"count" db:"count"`
}

type AppVersion struct {
	ID          int64     `db:"id" json:"-"`
	Version     string    `db:"version" json:"version"`
	MinVersion  string    `db:"min_version" json:"min_version"`
	Description string    `db:"description" json:"description"`
	Url         string    `db:"url" json:"url"`
	Cid         string    `db:"cid" json:"cid"`
	Size        int64     `db:"size" json:"size"`
	Platform    string    `db:"platform" json:"platform"`
	Lang        string    `db:"lang" json:"lang"`
	CreatedAt   time.Time `db:"created_at" json:"-"`
	UpdatedAt   time.Time `db:"updated_at" json:"-"`
}

type KOLLevelConfig struct {
	ID                      int64     `db:"id" json:"-"`
	Level                   int       `json:"level" db:"level"`
	CommissionPercent       float64   `db:"commission_percent" json:"commission_percent"`
	ParentCommissionPercent float64   `db:"parent_commission_percent" json:"parent_commission_percent"`
	Status                  int       `db:"status" json:"status"`
	DeviceThreshold         int       `db:"device_threshold" json:"device_threshold"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
}

type KOL struct {
	ID        int64     `db:"id" json:"-"`
	UserId    string    `json:"user_id" db:"user_id"`
	Level     int       `json:"level" db:"level"`
	Comment   string    `json:"comment" db:"comment"`
	Status    int       `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type KOLLevel struct {
	UserId                  string `db:"user_id"`
	Level                   int    `json:"level" db:"level"`
	ParentCommissionPercent int    `db:"parent_commission_percent" json:"parent_commission_percent"`
	ChildrenBonusPercent    int    `db:"children_bonus_percent" json:"children_bonus_percent"`
	DeviceThreshold         int64  `db:"device_threshold" json:"device_threshold"`
}

type ReferralCode struct {
	ID        int64     `db:"id" json:"-"`
	UserId    string    `json:"user_id" db:"user_id"`
	Code      string    `json:"code" db:"code"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"-"`
}

type ReferralCodeProfile struct {
	Code          string    `db:"code" json:"code"`
	ReferralUsers int       `db:"referral_users" json:"referral_users"`
	ReferralNodes int       `db:"referral_nodes" json:"referral_nodes"`
	EligibleNodes int       `db:"eligible_nodes" json:"eligible_nodes"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type KolLevelUpInfo struct {
	CurrenLevel             int     `json:"curren_level"`
	CommissionPercent       float64 `json:"commission_percent"`
	ParentCommissionPercent float64 `json:"parent_commission_percent"`
	ReferralNodes           int     `json:"referral_nodes"`
	LevelUpReferralNodes    int     `json:"level_up_referral_nodes"`
}

type ReferralRewardDaily struct {
	ReferrerUserId string    `json:"referrer_user_id" db:"referrer_user_id"`
	UserId         string    `json:"user_id" db:"user_id"`
	OnlineCount    int64     `json:"online_count" db:"online_count"`
	ReferrerReward float64   `json:"referrer_reward" db:"referrer_reward"`
	RefereeReward  float64   `json:"referee_reward" db:"referee_reward"`
	Time           time.Time `db:"time" json:"time"`
}

type DataCollectionEvent int

const (
	DataCollectionEventReferralCodePV = iota + 1
)

type DataCollection struct {
	Event     DataCollectionEvent `json:"event" db:"event"`
	Url       string              `json:"url" db:"url"`
	Os        string              `json:"os" db:"os"`
	Value     string              `json:"value" db:"value"`
	IP        string              `json:"ip" db:"ip"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
}

type DateValue struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Relationship int

const (
	RelationshipLevel1 = iota + 1
	RelationshipLevel2
)

type UserRewardDetail struct {
	UserId       string       `json:"user_id" db:"user_id"`
	FromUserId   string       `json:"from_user_id" db:"from_user_id"`
	Reward       float64      `json:"reward" db:"reward"`
	Relationship Relationship `json:"relationship" db:"relationship"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"-"`
}

type UserReward struct {
	UserId                string  `json:"user_id" db:"user_id"`
	Reward                float64 `json:"reward" db:"reward"`
	L2Reward              float64 `json:"l2_reward" db:"l2_reward"`
	L1Reward              float64 `json:"l1_reward" db:"l1_reward"`
	EligibleDeviceCount   int64   `json:"eligible_device_count" db:"eligible_device_count"`
	DeviceCount           int64   `json:"device_count" db:"device_count"`
	OnlineIncentiveReward float64 `json:"online_incentive_reward" db:"online_incentive_reward"`
}

type UserReferralRecord struct {
	UserId            string    `json:"user_id" db:"user_id"`
	ReferrerUserId    string    `json:"referrer_user_id" db:"referrer_user_id"`
	DeviceOnlineCount int64     `json:"device_online_count" db:"device_online_count"`
	Reward            float64   `json:"reward" db:"reward"`
	ReferrerReward    float64   `json:"referrer_reward" db:"referrer_reward"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type ReferralCounter struct {
	ReferralUsers  int64   `json:"referral_users" db:"referral_users"`
	ReferralNodes  int64   `json:"referral_nodes" db:"referral_nodes"`
	ReferrerReward float64 `json:"referrer_reward" db:"referrer_reward"`
	RefereeReward  float64 `json:"referee_reward" db:"referee_reward"`
}

// Test1NodeInfo 节点信息
type Test1NodeInfo struct {
	DeviceName    string  `db:"device_name" json:"deviceName"`        // 设备备注
	IP            string  `db:"external_ip" json:"ip"`                // 公网IP
	SystemVersion string  `db:"system_version" json:"systemVersion"`  // 程序版本
	DeviceID      string  `db:"device_id" json:"deviceId"`            // 设备id
	IPLocation    string  `db:"ip_location" json:"ipLocation"`        // IP所在区域
	TotalProfit   float64 `db:"cumulative_profit" json:"totalProfit"` // 累计收益
}

type PlainDeviceInfo struct {
	DeviceId         string `json:"device_id" db:"device_id"`
	DeviceName       string `json:"device_name" db:"device_name"`
	DeviceStatusCode int64  `json:"device_status_code" db:"device_status_code"`
	CumulativeProfit string `json:"cumulative_profit" db:"cumulative_profit"`
	NatType          string `json:"nat_type" db:"nat_type"`
	NodeType         string `json:"node_type" db:"node_type"`
	IPLocation       string `json:"ip_location" db:"ip_location"`
	ExternalIP       string `json:"external_ip" db:"external_ip"`
	SystemVersion    string `json:"system_version" db:"system_version"`
	IOSystem         string `json:"io_system" db:"io_system"`
}

type UserL1Reward struct {
	UserId    string    `json:"user_id" db:"user_id"`
	Reward    float64   `json:"reward" db:"reward"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type TotalUserStats struct {
	TotalRewards         float64 `json:"total_rewards" db:"total_rewards"`
	TotalNodeRewards     float64 `json:"total_node_rewards" db:"total_node_rewards"`
	TotalReferralRewards float64 `json:"total_referral_rewards" db:"total_referral_rewards"`
	TotalUsers           int64   `json:"total_users" db:"total_users"`
}

type TotalNodeStats struct {
	EdgeRewards      float64 `json:"edge_rewards" db:"edge_rewards"`
	CandidateRewards float64 `json:"candidate_rewards" db:"candidate_rewards"`
	// nodes
	OnlineNodes       int64 `json:"online_nodes" db:"online_nodes"`
	OnlineCandidates  int64 `json:"online_candidates" db:"online_candidates"`
	OnlineEdges       int64 `json:"online_edges" db:"online_edges"`
	OfflineNodes      int64 `json:"offline_nodes" db:"offline_nodes"`
	OfflineCandidates int64 `json:"offline_candidates" db:"offline_candidates"`
	OfflineEdges      int64 `json:"offline_edges" db:"offline_edges"`
	// ip
	TotalIPs   int64 `json:"total_ips" db:"total_ips"`
	OnlineIPs  int64 `json:"online_ips" db:"online_ips"`
	OfflineIPs int64 `json:"offline_ips" db:"offline_ips"`
	// cli
	OnlineVPSCLINodes  int64 `json:"online_vps_cli_nodes" db:"online_vps_cli_nodes"`
	OnlineWinCLINodes  int64 `json:"online_win_cli_nodes" db:"online_win_cli_nodes"`
	OnlineMacCLINodes  int64 `json:"online_mac_cli_nodes" db:"online_mac_cli_nodes"`
	OfflineVPSCLINodes int64 `json:"offline_vps_cli_nodes" db:"offline_vps_cli_nodes"`
	OfflineWinCLINodes int64 `json:"offline_win_cli_nodes" db:"offline_win_cli_nodes"`
	OfflineMacCLINodes int64 `json:"offline_mac_cli_nodes" db:"offline_mac_cli_nodes"`
	// client
	OnlineWinNodes  int64 `json:"online_win_nodes" db:"online_win_nodes"`
	OnlineMacNodes  int64 `json:"online_mac_nodes" db:"online_mac_nodes"`
	OnlineAppNodes  int64 `json:"online_app_nodes" db:"online_app_nodes"`
	OfflineWinNodes int64 `json:"offline_win_nodes" db:"online_win_nodes"`
	OfflineMacNodes int64 `json:"offline_mac_nodes" db:"online_mac_nodes"`
	OfflineAppNodes int64 `json:"offline_app_nodes" db:"online_app_nodes"`
	// storage
	TitanStorage     float64 `json:"titan_disk_space" db:"titan_disk_space"`
	TitanStorageUsed float64 `json:"titan_disk_usage" db:"titan_disk_usage"`
}

type TotalAssetStats struct {
	// download
	DownloadCount           int64 `json:"download_count" db:"download_count"`
	DownloadSize            int64 `json:"download_size" db:"download_size"`
	SuccessfulDownloadCount int64 `json:"successful_download_count" db:"successful_download_count"`
	FailedDownloadCount     int64 `json:"failed_download_count" db:"failed_download_count"`
	DownloadSpeed           int64 `json:"download_speed" db:"download_speed"`

	// upload
	UploadCount           int64 `json:"upload_count" db:"upload_count"`
	UploadSize            int64 `json:"upload_size" db:"upload_size"`
	SuccessfulUploadCount int64 `json:"successful_upload_count" db:"successful_upload_count"`
	FailedUploadCount     int64 `json:"failed_upload_count" db:"failed_upload_count"`
	UploadSpeed           int64 `json:"upload_speed" db:"upload_speed"`
}

type NodeIPHistory struct {
	NodeId     string `json:"node_id" db:"node_id"`
	ExternalIP string `json:"external_ip" db:"external_ip"`
	Date       string `json:"date" db:"date"`
}

type AreaConfig struct {
	AreaId string `json:"area_id" db:"area_id"`
	NameCn string `json:"name_cn" db:"name_cn"`
	NameEn string `json:"name_en" db:"name_en"`
}

type QualitiesNode struct {
	NodeId                 string  `json:"node_id" db:"node_id"`
	CacheCount             int64   `db:"cache_count" json:"cache_count"`
	RetrievalCount         int64   `db:"retrieval_count" json:"retrieval_count"`
	AssetSucceededCount    int64   `db:"asset_succeeded_count" json:"asset_succeeded_count"`
	AssetFailedCount       int64   `db:"asset_failed_count" json:"asset_failed_count"`
	RetrieveSucceededCount int64   `db:"retrieve_succeeded_count" json:"retrieve_succeeded_count"`
	RetrieveFailedCount    int64   `db:"retrieve_failed_count" json:"retrieve_failed_count"`
	BandwidthUp            float64 `db:"bandwidth_up" json:"bandwidth_up"`
	BandwidthDown          float64 `db:"bandwidth_down" json:"bandwidth_down"`
}

type WorkerdNode struct {
	NodeId                string `db:"node_id" json:"node_id"`
	ProjectCount          int64  `db:"project_count" json:"project_count"`
	ProjectSucceededCount int64  `db:"project_succeeded_count" json:"project_succeeded_count"`
	ProjectFailedCount    int64  `db:"project_failed_count" json:"project_failed_count"`
}

type IPNodeCount struct {
	ExternalIP      string `json:"external_ip" db:"external_ip"`
	Area            string `json:"area"  db:"area_id"`
	OnlineNodeCount int64  `json:"online_node_count"  db:"online_node_count"`
	TotalNodeCount  int64  `json:"total_node_count"  db:"total_node_count"`
}
