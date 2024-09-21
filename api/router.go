package api

import (
	"bytes"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/config"
	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("api")
)

func RegisterRouters(route *gin.Engine, cfg config.Config) {
	RegisterRouterWithJWT(route, cfg)
	RegisterRouterWithAPIKey(route)
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if strings.Contains(c.Request.URL.Path, "storage") {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			body, _ := io.ReadAll(tee)
			c.Request.Body = io.NopCloser(&buf)
			if string(body) != "" {
				log.Debug(string(body))
			}
		}
		//log.Debug(c.Request.Header)
		c.Next()
	}
}

// var ignoreRouterMap = map[string]bool{
// 	"/api/v1/user/ads/banners": true,
// 	"/api/v2/app_version":      true,
// 	"/api/v2/device":           true,
// }

// func IgnoreSpecificRouter(c *gin.Context) {
// 	if ignoreRouterMap[c.Request.URL.Path] {
// 		c.Next()

// 	}
// }

func RegisterRouterWithJWT(router *gin.Engine, cfg config.Config) {
	apiV1 := router.Group("/api/v1")
	apiV2 := router.Group("/api/v2")
	link := router.Group("/link")

	authMiddleware, err := jwtGinMiddleware(cfg.SecretKey)
	if err != nil {
		log.Fatalf("jwt auth middleware: %v", err)
	}

	err = authMiddleware.MiddlewareInit()
	if err != nil {
		log.Fatalf("authMiddleware.MiddlewareInit: %v", err)
	}

	// testnet
	apiV2.POST("/subscribe", SubscribeHandler)

	// dashboard
	// Deprecated: use /height instead
	apiV2.GET("/location", GetIPLocationHandler)
	apiV2.GET("/get_high", GetBlockHeightHandler)
	apiV2.GET("/height", GetBlockHeightHandler)
	apiV2.GET("/all_areas", GetAllAreas)
	apiV2.GET("/get_index_info", GetIndexInfoHandler)
	apiV2.GET("/get_query_info", GetQueryInfoHandler)
	apiV2.GET("/releases", GetReleasesHandler)
	apiV2.GET("/app_version", GetAppVersionHandler)
	apiV2.POST("/device", GetDeviceProfileHandler)
	apiV2.POST("/plain/device/info", GetPlainDeviceInfoHandler)
	apiV2.POST("/device/binding", DeviceBindingHandler)
	apiV2.GET("/device/query_code", QueryDeviceCodeHandler)
	apiV2.GET("/device/distribution", GetDeviceDistributionHandler)
	apiV2.POST("/data/collection", DataCollectionHandler)
	apiV2.GET("/acme", AcmeHandler)
	// index info all nodes info from device info
	apiV2.GET("/get_nodes_info", GetNodesInfoHandler)
	apiV2.GET("/get_device_info", GetDeviceInfoHandler)
	apiV2.GET("/get_device_status", GetDeviceStatusHandler)
	apiV2.GET("/get_map_info", GetMapInfoHandler)
	apiV2.GET("/get_device_info_daily", GetDeviceInfoDailyHandler)
	apiV2.GET("/get_diagnosis_days", GetDeviceDiagnosisDailyByDeviceIdHandler)
	// by-user_id or all node count
	apiV2.GET("/get_diagnosis_days_user", GetDeviceDiagnosisDailyByUserIdHandler)
	apiV2.GET("/get_diagnosis_hours", GetDeviceDiagnosisHourHandler)
	apiV2.GET("/get_cache_hours", GetCacheHourHandler)
	apiV2.GET("/get_cache_days", GetCacheDaysHandler)

	// node daily count
	apiV2.GET("/get_nodes_days", GetDiskDaysHandler)
	apiV2.GET("/node_online_incentive", GetDeviceOnlineIncentivesHandler)

	// request from titan api
	apiV2.GET("/get_cache_list", GetCacheListHandler)
	apiV2.GET("/get_retrieval_list", GetRetrievalListHandler)
	apiV2.GET("/get_validation_list", GetValidationListHandler)
	apiV2.GET("/get_replica_list", GetReplicaListHandler)
	apiV2.GET("/get_profit_details", GetProfitDetailsHandler)
	apiV2.GET("/login_before", GetNonceStringHandler)
	apiV2.POST("/login", authMiddleware.LoginHandler)
	apiV2.POST("/logout", authMiddleware.LogoutHandler)
	apiV2.GET("/get_user_device_count", GetUserDevicesCountHandler)

	apiV2.Use(authMiddleware.MiddlewareFunc())
	apiV2.Use(AuthRequired(authMiddleware))
	// console
	apiV2.GET("/device_update", DeviceUpdateHandler)
	apiV2.POST("/device_update", DeviceUpdateHandler)
	apiV2.GET("/get_device_info_auth", GetDeviceInfoHandler)
	apiV2.GET("/device_unbinding", DeviceUnBindingHandlerOld)
	apiV2.GET("/get_user_device_profile", GetUserDeviceProfileHandler)
	apiV2.GET("/get_device_active_info", GetDeviceActiveInfoHandler)
	apiV2.POST("/wallet/bind", BindWalletHandler)
	apiV2.POST("/wallet/unbind", UnBindWalletHandler)
	apiV2.GET("/referral_list", GetReferralListHandler)
	apiV2.GET("/generate/code", GenerateCodeHandler)

	// user
	user := apiV1.Group("/user")
	user.POST("/register", UserRegister)
	user.POST("/password_reset", PasswordRest)
	user.GET("/captcha/block", GetBlockCaptcha)
	user.POST("/verify_code", GetNumericVerifyCodeHandler)
	user.POST("/login", authMiddleware.LoginHandler)
	user.POST("/logout", authMiddleware.LogoutHandler)
	user.GET("/ads/banners", GetBannersHandler)
	user.GET("/ads/notices", GetNoticesHandler)
	user.GET("/ads/history", GetAdsHistoryHandler)
	user.GET("ads/click", AdsClickIncrHandler)
	user.POST("/upload", FileUploadHandler)
	user.POST("/bugs/report", BugReportHandler)
	user.GET("/bugs/list", MyBugReportListHandler)
	user.GET("/locators", LocatorFromConfigHandler)
	user.POST("/edge/batch/report", BatchReportHandler)
	user.GET("/edge/batch/address", UserBatchAddressHandler)
	user.GET("/edge/config", GetEdgeConfigHandler)
	user.POST("/edge/config", SetEdgeConfigHandler)
	user.Use(authMiddleware.MiddlewareFunc())
	user.GET("/refresh_token", authMiddleware.RefreshHandler)
	user.POST("/info", GetUserInfoHandler)
	user.POST("/referral_code/new", AddReferralCodeHandler)
	user.GET("/referral_code/detail", GetReferralCodeDetailHandler)
	user.GET("/referral_code/stat", GetReferralCodeStatHandler)

	// admin
	admin := apiV1.Group("/admin")
	adminMiddleware := *authMiddleware
	adminMiddleware.Authorizator = AdminOnly
	err = adminMiddleware.MiddlewareInit()
	if err != nil {
		log.Fatal(err)
	}

	admin.Use(adminMiddleware.MiddlewareFunc())
	admin.GET("/get_login_log", GetLoginLogHandler)
	admin.GET("/get_operation_log", GetOperationLogHandler)
	admin.GET("/get_node_daily_trend", GetNodeDailyTrendHandler)
	admin.GET("/kol/list", GetKOLListHandler)
	admin.POST("/kol/add", AddKOLHandler)
	admin.POST("/kol/update", UpdateKOLHandler)
	admin.POST("/kol/delete", DeleteKOLHandler)
	admin.POST("/kol_level/add", AddKOLLevelHandler)
	admin.GET("/kol_level/list", GetKOLLevelConfigHandler)
	admin.POST("/kol_level/update", UpdateKOLLevelHandler)
	admin.POST("/kol_level/delete", DeleteKOLLevelHandler)
	admin.GET("/referral_reward_daily", GetReferralRewardDailyHandler)
	admin.GET("/referral_reward_daily/export", ExportReferralRewardDailyHandler)
	// ads
	admin.GET("/ads/list", ListAdsHandler)
	admin.POST("/ads/add", AddAdsHandler)
	admin.POST("/ads/delete", DeleteAdsHandler)
	admin.POST("/ads/update", UpdateAdsHandler)
	admin.POST("/upload", FileUploadHandler)
	// bugs
	admin.GET("/bugs/list", BugReportListHandler)
	admin.POST("/bugs/edit", BugEditHandler)
	// acme
	admin.POST("/acme/add", AcmeAddHandler)
	// batch
	admin.GET("/batch/edge", BatchGetHandler)
	admin.DELETE("/batch/edge", BatchDelHandler)
	admin.POST("/batch/edge", BatchReportHandler)
	admin.POST("/batch/address", BatchAddressSetHandler)
	admin.GET("/batch/address", BatchAddressListHandler)
	admin.DELETE("/batch/address", BatchAddressDelHandler)

	// dashboards
	admin.GET("/areas", GetAreasHandler)
	admin.GET("/total_stats", GetTotalStatsHandler)
	admin.GET("/ip_changed_records", GetNodeIPChangedRecordsHandler)
	admin.GET("/asset_records", GetAssetRecordsHandler)
	admin.GET("/successful_replicas", GetSuccessfulReplicasHandler)
	admin.GET("/failed_replicas", GetFailedReplicasHandler)
	admin.GET("/workerd_nodes", GetWorkerdNodesHandler)
	admin.GET("/qualities_nodes", GetQualitiesNodesHandler)
	admin.GET("/project/overview", GetProjectOverviewHandler)
	admin.GET("/project/info", GetProjectInfoHandler)
	admin.GET("/ip_records", GetIPRecordsHandler)

	// storage
	storage := apiV1.Group("/storage")
	storage.Use(gin.Logger())
	storage.GET("/get_map_info", GetMapInfoHandler)
	// Deprecated: use /user/verify_code instead
	storage.POST("/get_verify_code", GetNumericVerifyCodeHandler)
	// Deprecated: use /user/register instead
	storage.POST("/register", UserRegister)
	// Deprecated: use /user/password_reset instead
	storage.POST("/password_reset", PasswordRest)
	storage.GET("/login_before", GetNonceStringHandler)
	storage.POST("/login", authMiddleware.LoginHandler)
	storage.POST("/logout", authMiddleware.LogoutHandler)
	link.GET("/", GetShareLinkHandler)
	storage.GET("/get_link", ShareLinkHandler)
	storage.GET("/create_link", CreateShareLinkHandler)
	storage.GET("/share_need_pass", ShareNeedPassHandler)
	storage.POST("/check_share", CheckShareLinkHandler)
	storage.GET("/get_map_cid", GetMapByCidHandler)
	storage.GET("/get_map_link", GetShareLinkHandler)
	storage.GET("/get_asset_detail", GetAssetDetailHandler)
	storage.GET("/get_asset_location", GetLocationHandler)
	storage.GET("/monitor", GetMonitor)
	// storage.GET("/file_pass_nonce", FilePassNonceHandler)
	storage.GET("/get_asset_status", GetAssetStatusHandler)
	storage.GET("/get_open_asset_status", GetOpenAssetStatusHandler)
	storage.GET("/get_fil_storage_list", GetFilStorageListHandler)
	storage.GET("/get_area_id", GetSchedulerAreaIDs)
	storage.GET("/temp_file/get_upload_file", UploadTempFileCar)
	storage.POST("/temp_file/upload", UploadTmepFile)
	storage.GET("/temp_file/info/:cid", GetUploadInfo)
	storage.GET("/temp_file/share/:cid", ShareTempFile)
	storage.GET("/temp_file/download/:cid", DownloadTempFile)
	// storage.Use(authMiddleware.MiddlewareFunc())
	storage.GET("/open_asset", OpenAssetHandler) // 打开公共的文件，需要统计访问次数
	storage.POST("/sync_data", SyncHourData)
	storage.GET("/count", GetStorageCount)
	storage.Use(AuthRequired(authMiddleware))
	storage.GET("/share_before", ShareBeforeHandler)
	storage.GET("/share_asset", ShareAssetsHandler)

	storage.GET("/share_link_info", ShareLinkInfoHandler)
	storage.POST("/share_link_update", ShareLinkUpdateHandler)

	storage.GET("/get_locateStorage", GetAllocateStorageHandler)
	storage.GET("/get_storage_size", GetStorageSizeHandler) // 获取用户存储空间信息
	storage.GET("/get_vip_info", GetUserVipInfoHandler)     // 判断用户是否为vip
	storage.GET("/get_user_access_token", GetUserAccessTokenHandler)
	storage.GET("/get_upload_info", GetUploadInfoHandler)
	storage.GET("/create_asset", CreateAssetHandler)
	storage.POST("/create_asset", CreateAssetPostHandler)
	storage.POST("/import_from_ipfs", CreateAssetFromIPFSHandler)
	storage.POST("/export_to_ipfs", ExportAssetToIPFSHandler)
	storage.GET("/delete_asset", DeleteAssetHandler)
	storage.GET("/get_asset_info", GetAssetInfoHandler)
	storage.GET("/get_asset_list", GetAssetListHandler)
	storage.GET("/get_all_asset_list", GetAssetAllListHandler)
	storage.GET("/share_status_set", UpdateShareStatusHandler) // 修改分享状态
	storage.GET("/create_key", CreateKeyHandler)               // TODO: 需要讨论key生成方式
	storage.GET("/get_keys", GetKeyListHandler)
	storage.GET("/delete_key", DeleteKeyHandler)
	storage.GET("/get_asset_count", GetAssetCountHandler)
	storage.GET("/get_user_info_hour", GetStorageHourV2Handler)
	storage.GET("/get_user_info_daily", GetStorageDailyHandler)
	storage.GET("/refresh_token", authMiddleware.RefreshHandler)
	storage.GET("/new_secret", CreateNewSecretKeyHandler)
	storage.GET("/get_key_perms", GetAPIKeyPermsHandler) // 获取 key 的权限
	storage.GET("/create_group", CreateGroupHandler)     // 创建文件夹
	storage.GET("/get_groups", GetGroupsHandler)         // 获取文件夹信息
	storage.GET("/get_asset_group_list", GetAssetGroupListHandler)
	storage.GET("/delete_group", DeleteGroupHandler)
	storage.POST("/rename_group", RenameGroupHandler)
	storage.GET("/move_group_to_group", MoveGroupToGroupHandler)
	storage.GET("/move_asset_to_group", MoveAssetToGroupHandler)
	storage.POST("/move_node", MoveNode)
	// storage.POST("/ipfs_info", GetIPFSInfoByCIDs)

	//signature
	signature := apiV1.Group("/sign")
	signature.GET("/info", getSignInfo)
	signature.GET("/summary", getSummaryInfo)

	signature.POST("/upload", setSignInfo)
	signature.POST("/command", getCommand)

	// url
	uri := apiV1.Group("/url")
	uri.GET("/discord", getDiscordURL)

	// test1
	test1 := apiV1.Group("/test1")
	test1.Use(authMiddleware.MiddlewareFunc())
	tnc := &Test1NodeController{}
	// test1.GET("/node/info", tnc.GetNodes)
	// test1.PUT("/node/update_name", tnc.UpdateDeviceName)
	test1.GET("/node/nums", tnc.GetNodeNums)
	test1.PUT("/node/delete_offline", tnc.DeleteOffLineNode)
	test1.PUT("/node/move_back_deleted", tnc.MoveBackDeletedNode)

	apiV1.GET("/country_count", GetCountryCount)
}

func RegisterRouterWithAPIKey(router *gin.Engine) {
	authV1 := router.Group("/v1")
	storage := authV1.Group("/storage")
	storage.Use(AuthAPIKeyMiddlewareFunc())
	storage.POST("/add_fil_storage", CreateFilStorageHandler)
	storage.GET("/backup_assets", GetBackupAssetsHandler)
	storage.POST("/backup_result", BackupResultHandler)

	app := authV1.Group("/app")
	app.Use(AuthAPIKeyMiddlewareFunc())
	app.POST("/new_version", CreateAppVersionHandler)
	app.POST("/new_release", UpdateReleaseInfoHandler)
}
