package statistics

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/golang-module/carbon/v2"
	errs "github.com/pkg/errors"
	"sort"
	"time"
)

const (
	DefaultCommissionPercent = 5
)

const batchSize = 1000

// AddDeviceInfoHours 写入 device_info_hour 表
func AddDeviceInfoHours(ctx context.Context, upsertDevice []*model.DeviceInfoHour) error {
	log.Info("start to fetch device info hours")
	start := time.Now()
	defer func() {
		log.Infof("fetch device info hours done, cost: %v", time.Since(start))
	}()

	err := dao.BulkUpsertDeviceInfoHours(ctx, upsertDevice)
	if err != nil {
		log.Errorf("bulk upsert device info: %v", err)
	}

	if start.Minute() != 0 {
		return nil
	}

	// Add a redundant record to make it easier to count data within the range of 0-60 minutes
	for i := 0; i < len(upsertDevice); i++ {
		upsertDevice[i].Time = start.Add(-1 * time.Minute)
	}

	err = dao.BulkUpsertDeviceInfoHours(ctx, upsertDevice)
	if err != nil {
		log.Errorf("bulk upsert device info: %v", err)
	}

	return nil
}

func GetDeviceUserId(ctx context.Context, deviceId string) string {
	deviceOrdinaryInfo, err := dao.GetDeviceInfo(ctx, deviceId)
	if err != nil {
		log.Errorf("get device info: %v", err)
		return ""
	}

	if deviceOrdinaryInfo.UserID != "" {
		err = dao.SetDeviceUserIdToCache(ctx, deviceId, deviceOrdinaryInfo.UserID, deviceOrdinaryInfo.AreaID)
		if err != nil {
			log.Errorf("set device user to cahce: %v", err)
		}

		return deviceOrdinaryInfo.UserID
	}

	signature, err := dao.GetSignatureByNodeId(ctx, deviceId)
	if err != nil {
		log.Errorf("get signatrue by node id: %v", err)
		//return ""
	}

	if signature == nil {
		return ""
	}

	if err = dao.UpdateUserDeviceInfo(ctx, &model.DeviceInfo{
		UserID:     signature.Username,
		DeviceID:   deviceId,
		BindStatus: "binding",
	}); err != nil {
		log.Errorf("update device binding status: %v", err)
	}

	return signature.Username
}

func QueryDataByDate(DateFrom, DateTo string) []map[string]string {
	sqlClause := fmt.Sprintf("select device_id, sum(income) as income,online_time from device_info_daily "+
		"where  time>='%s' and time<='%s' group by device_id;", DateFrom, DateTo)
	if DateFrom == "" {
		sqlClause = fmt.Sprintf("select device_id, sum(income) as income,online_time from device_info_daily " +
			" group by device_id;")
	}
	data, err := dao.GetQueryDataList(sqlClause)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	return data
}

func (s *Statistic) SumDeviceInfoDaily() error {
	log.Info("start to sum device info daily")
	start := time.Now()
	defer func() {
		log.Infof("sum device info daily done, cost: %v", time.Since(start))
	}()

	if err := s.calculateAndUpsertDailyInfo(); err != nil {
		log.Errorf("calculate and upsert daily info: %v", err)
		return err
	}

	return nil
}

func (s *Statistic) calculateAndUpsertDailyInfo() error {
	startOfTodayTime := carbon.Now().StartOfDay().String()
	endOfTodayTime := carbon.Now().Tomorrow().StartOfDay().String()

	total, err := s.countDeviceInfoHour(startOfTodayTime, endOfTodayTime)
	if err != nil {
		return errs.Wrap(err, "count device info hour")
	}

	where := fmt.Sprintf("where time>='%s' and time<='%s'", startOfTodayTime, endOfTodayTime)
	sqlClause := fmt.Sprintf(`select i.user_id, i.device_id, date_format(i.time, '%%Y-%%m-%%d') as date, 
			i.nat_ratio, i.disk_usage, i.disk_space,i.latency, i.pkg_loss_ratio, i.bandwidth_up, i.bandwidth_down, 
			max(i.hour_income) - min(i.hour_income) as hour_income,
			max(i.online_time) - min(i.online_time) as online_time,
			max(i.upstream_traffic) - min(i.upstream_traffic) as upstream_traffic,
			max(i.downstream_traffic) - min(i.downstream_traffic) as downstream_traffic,
			max(i.retrieval_count) - min(i.retrieval_count) as retrieval_count,
			max(i.block_count) - min(i.block_count) as block_count
			from (select * from device_info_hour %s order by id desc limit %d) i                                                                                   
			group by device_id`, where, total)

	dataList, err := dao.GetQueryDataList(sqlClause)
	if err != nil {
		return errs.Wrap(err, "get query data list")
	}

	dailyInfos, err := s.buildDailyInfos(dataList)
	if err != nil {
		return errs.Wrap(err, "build daily infos")
	}

	err = dao.BulkUpsertDeviceInfoDaily(context.Background(), dailyInfos)
	if err != nil {
		return errs.Wrap(err, "bulk upsert device info daily")
	}

	return nil
}

func (s *Statistic) countDeviceInfoHour(start, end string) (int64, error) {
	where := fmt.Sprintf("where time>='%s' and time<='%s'", start, end)
	var total int64
	err := dao.DB.GetContext(s.ctx, &total, fmt.Sprintf(`SELECT count(*) FROM %s %s`, "device_info_hour", where))
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Statistic) buildDailyInfos(dataList []map[string]string) ([]*model.DeviceInfoDaily, error) {
	var dailyInfos []*model.DeviceInfoDaily
	for _, data := range dataList {
		var daily model.DeviceInfoDaily
		daily.Time, _ = time.Parse(formatter.TimeFormatDateOnly, data["date"])
		daily.DiskUsage = formatter.Str2Float64(data["disk_usage"])
		daily.DiskSpace = formatter.Str2Float64(data["disk_space"])
		daily.NatRatio = formatter.Str2Float64(data["nat_ratio"])
		daily.Income = formatter.Str2Float64(data["hour_income"])
		daily.OnlineTime = formatter.Str2Float64(data["online_time"])
		if daily.OnlineTime > 1440 {
			daily.OnlineTime = 1440
		}
		daily.UpstreamTraffic = formatter.Str2Float64(data["upstream_traffic"])
		daily.DownstreamTraffic = formatter.Str2Float64(data["downstream_traffic"])
		daily.RetrievalCount = formatter.Str2Int64(data["retrieval_count"])
		daily.BlockCount = formatter.Str2Int64(data["block_count"])
		daily.PkgLossRatio = formatter.Str2Float64(data["pkg_loss_ratio"])
		daily.Latency = formatter.Str2Float64(data["latency"])
		daily.DeviceID = data["device_id"]
		daily.UserID = data["user_id"]
		daily.BandwidthUp = formatter.Str2Float64(data["bandwidth_up"])
		daily.BandwidthDown = formatter.Str2Float64(data["bandwidth_down"])
		daily.UserID = data["user_id"]
		daily.CreatedAt = time.Now()
		daily.UpdatedAt = time.Now()
		dailyInfos = append(dailyInfos, &daily)
	}
	return dailyInfos, nil
}

func SumDeviceInfoProfit() error {
	log.Info("start to sum device info profit")
	start := time.Now()
	defer func() {
		log.Infof("sum device info profit done, cost: %v", time.Since(start))
	}()

	updatedDevices := make(map[string]*model.DeviceInfo)

	updateDeviceInfoForTimeRange(updatedDevices, carbon.Yesterday(), carbon.Yesterday(), "YesterdayProfit")
	updateDeviceInfoForTimeRange(updatedDevices, carbon.Now().SubDays(6), carbon.Now(), "SevenDaysProfit")
	updateDeviceInfoForTimeRange(updatedDevices, carbon.Now().SubDays(29), carbon.Now(), "MonthProfit")
	updateDeviceInfoForTimeRange(updatedDevices, carbon.Now(), carbon.Now(), "TodayProfit", "TodayOnlineTime")

	var count int
	deviceInfos := make([]*model.DeviceInfo, 0)
	for _, deviceInfo := range updatedDevices {
		count++

		deviceInfos = append(deviceInfos, deviceInfo)

		if len(deviceInfos) == 1000 || count == len(updatedDevices) {
			if err := dao.BulkUpdateDeviceInfo(context.Background(), deviceInfos); err != nil {
				log.Errorf("bulk update devices: %v", err)
			}
			deviceInfos = make([]*model.DeviceInfo, 0)
		}
	}

	return nil
}

func updateDeviceInfoForTimeRange(updatedDevices map[string]*model.DeviceInfo, start, end carbon.Carbon, profitFields ...string) {
	data := QueryDataByDate(start.StartOfDay().String(), end.EndOfDay().String())
	for _, d := range data {
		deviceID := d["device_id"]
		device, ok := updatedDevices[deviceID]
		if !ok {
			device = &model.DeviceInfo{DeviceID: deviceID}
			updatedDevices[deviceID] = device
		}

		for _, field := range profitFields {
			switch field {
			case "YesterdayProfit":
				updatedDevices[deviceID].YesterdayProfit = formatter.Str2Float64(d["income"])
				updatedDevices[deviceID].YesterdayOnlineTime = formatter.Str2Float64(d["online_time"])
			case "SevenDaysProfit":
				updatedDevices[deviceID].SevenDaysProfit = formatter.Str2Float64(d["income"])
			case "MonthProfit":
				updatedDevices[deviceID].MonthProfit = formatter.Str2Float64(d["income"])
			case "TodayProfit":
				updatedDevices[deviceID].TodayProfit = formatter.Str2Float64(d["income"])
				updatedDevices[deviceID].TodayOnlineTime = formatter.Str2Float64(d["online_time"])
			}
		}
	}
}

// SumUserReward
// 奖励规则
// - 普通人： 邀请好友获得积分时，您将获得他们积分的10%作为奖励。
// - KOL-1级： 邀请好友获得积分时，您将获得他们积分的15%作为奖励。
// - KOL-2级： 邀请好友获得积分时，您将获得他们积分的20%作为奖励。
// - KOL-3级： 邀请好友获得积分时，您将获得他们积分的30%作为奖励。
// 二级邀请奖励
// - 当您的好友（被您邀请的人）邀请其他人并获得积分时，您也将获得该二级好友积分的部分奖励：
// - 普通人： 获得二级好友积分的5%。
// - KOL-1级： 获得二级好友积分的7.5%。
// - KOL-2级： 获得二级好友积分的10%。
// - KOL-3级： 获得二级好友积分的15%。
func SumUserReward(ctx context.Context) error {
	log.Info("start to sum user reward")
	start := time.Now()
	defer func() {
		log.Infof("sum user reward done, cost: %v", time.Since(start))
	}()

	// 计算所有用户的L2累计收益, 当天收益, 满足条件的节点数量
	userRewardSum, err := dao.SumAllUsersReward(ctx, config.Cfg.EligibleOnlineMinutes)
	if err != nil {
		log.Errorf("SumAllUsersReward: %v", err)
		return err
	}

	// 获取所有的邀请关系
	referrerUserIdInUser, err := dao.GetAllUserReferrerUserId(ctx)
	if err != nil {
		log.Errorf("GetAllUserReferrerUserId: %v", err)
		return err
	}

	// 计算满足条件的邀请节点数量
	sumReferrerDeviceCount := make(map[string]int64)
	for _, ur := range userRewardSum {
		if parentId, existing := referrerUserIdInUser[ur.UserId]; existing {
			sumReferrerDeviceCount[parentId] += ur.EligibleDeviceCount
		}
	}

	kolConfig, _, err := dao.GetKolLevelConfig(ctx, dao.QueryOption{})
	if err != nil {
		return err
	}

	kolConfigMap := make(map[int]*model.KOLLevelConfig)
	for _, k := range kolConfig {
		kolConfigMap[k.Level] = k
	}

	// 通过邀请的节点数量, 计算用户的当前的等级
	userLevel, err := applyUserLevel(ctx, kolConfig, sumReferrerDeviceCount)
	if err != nil {
		return err
	}

	referralReward := make(map[string]float64)
	var updateDetails []*model.UserRewardDetail

	var userRewards []*model.UserReward
	for _, userReward := range userRewardSum {
		// 今日收益有变动的
		//if userReward.Reward == 0 {
		//	continue
		//}

		user, uErr := dao.GetUserByUsername(ctx, userReward.UserId)
		if uErr != nil {
			return uErr
		}

		l1Rw, err := dao.GetUserL1Reward(ctx, userReward.UserId)
		if err != nil {
			return err
		}

		if user == nil {
			log.Errorf("user not found: %s", userReward.UserId)
			continue
		}

		// 计算收益的增值, 在有惩罚机制的情况下, 收益有可能是负数
		changedRewards := userReward.L2Reward - (user.Reward - l1Rw.Reward)
		changedDeviceCounts := userReward.DeviceCount - user.DeviceCount

		if changedRewards != 0 || changedDeviceCounts != 0 {
			userRewards = append(userRewards, userReward)
		}

		if changedRewards == 0 {
			continue
		}

		// 计算一级邀请奖励
		if user.ReferrerUserId != "" {
			// 当前邀请人的等级
			currentRefUserLevel := userLevel[user.ReferrerUserId]
			// 获取当前邀请人等级的邀请奖励比例
			currentLevelConfig, existing := kolConfigMap[currentRefUserLevel]
			if !existing {
				log.Errorf("user level not existing, username: %s, level: %d", user.ReferrerUserId, currentRefUserLevel)
			}

			rw := changedRewards * currentLevelConfig.CommissionPercent / 100
			referralReward[user.ReferrerUserId] += rw
			parent, pErr := dao.GetUserByUsername(ctx, user.ReferrerUserId)
			if pErr != nil {
				return pErr
			}

			updateDetails = append(updateDetails, &model.UserRewardDetail{
				UserId:       user.ReferrerUserId,
				FromUserId:   userReward.UserId,
				Reward:       rw,
				Relationship: model.RelationshipLevel1,
			})

			// 计算二级的奖励
			if parent.ReferrerUserId != "" {
				// 当前邀请人的等级
				currentParentRefUserLevel := userLevel[parent.ReferrerUserId]
				// 获取当前邀请人等级的邀请奖励比例
				currentParentLevelConfig, pExisting := kolConfigMap[currentParentRefUserLevel]
				if !pExisting {
					log.Errorf("user parent level not existing, username: %s, level: %d", parent.ReferrerUserId, currentParentRefUserLevel)
				}

				prw := changedRewards * currentParentLevelConfig.ParentCommissionPercent / 100

				referralReward[parent.ReferrerUserId] += prw

				updateDetails = append(updateDetails, &model.UserRewardDetail{
					UserId:       parent.ReferrerUserId,
					FromUserId:   userReward.UserId,
					Reward:       prw,
					Relationship: model.RelationshipLevel2,
				})
			}
		}
	}

	log.Infof("today user reward changed count: %d %d %d", len(userRewards), len(referralReward), len(updateDetails))

	// 更新 user_l1_reward
	if err = updateUserL1Rewards(ctx, userRewards); err != nil {
		return err
	}

	// 更新 users 表, 用户的 reward, device_count, online_incentive_reward
	if err = updateUserRewards(ctx, userRewards); err != nil {
		return err
	}

	// 更新 users 表, 邀请人的 referral_reward
	err = updateReferrerReward(ctx, referralReward)
	if err != nil {
		return err
	}

	// 更新邀请奖励详情
	err = updateUserRewardDetails(ctx, updateDetails)
	if err != nil {
		return err
	}

	return nil
}

func updateUserRewards(ctx context.Context, userRewards []*model.UserReward) error {
	var todos []*model.User

	for i, u := range userRewards {
		todos = append(todos, &model.User{
			Username:              u.UserId,
			Reward:                u.L2Reward + u.L1Reward,
			DeviceCount:           u.DeviceCount,
			EligibleDeviceCount:   u.EligibleDeviceCount,
			OnlineIncentiveReward: u.OnlineIncentiveReward,
		})

		// Perform bulk insert when todos reaches a multiple of batchSize or at the end of updateUserRewards
		if len(todos)%batchSize == 0 || i == len(userRewards)-1 {
			if err := dao.BulkUpdateUserReward(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserReward")
			}
			todos = todos[:0] // Reset todos slice without reallocating memory
		}
	}

	return nil
}

func updateReferrerReward(ctx context.Context, referralReward map[string]float64) error {
	var todos []*model.User
	var i int

	for userId, rw := range referralReward {
		todos = append(todos, &model.User{Username: userId, ReferralReward: rw})

		i++
		if len(todos)%batchSize == 0 || i == len(referralReward) {
			if err := dao.BulkUpdateUserReferralReward(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserReferralReward")
			}
			todos = todos[:0]
		}

	}

	return nil
}

func updateUserRewardDetails(ctx context.Context, updateDetails []*model.UserRewardDetail) error {
	var todos []*model.UserRewardDetail

	for i, u := range updateDetails {
		todos = append(todos, u)

		// Perform bulk insert when todos reaches a multiple of batchSize or at the end of updateUserRewards
		if len(todos)%batchSize == 0 || i == len(updateDetails)-1 {
			if err := dao.BulkUpdateUserRewardDetails(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserRewardDetails")
			}
			todos = todos[:0] // Reset todos slice without reallocating memory
		}
	}

	return nil
}

// 升级条件
// - 普通人： 邀请有效节点数量低于200。
// - KOL - 1级： 邀请有效节点数量在 200-1000 之间。
// - KOL - 2级： 邀请有效节点数量在 1000-2000 之间。
// - KOL - 3级： 邀请有效节点数量超过2000。
func applyUserLevel(ctx context.Context, kolConfig []*model.KOLLevelConfig, sumReferrerDeviceCount map[string]int64) (map[string]int, error) {
	var updateKols []*model.KOL
	out := make(map[string]int)

	sort.Slice(kolConfig, func(i, j int) bool {
		return kolConfig[i].Level < kolConfig[j].Level
	})

	adminAddedKols, err := dao.GetAdminAddedKolLevel(ctx)
	if err != nil {
		return nil, err
	}

	for userId, eligibleCount := range sumReferrerDeviceCount {
		kol := &model.KOL{
			UserId:  userId,
			Status:  1,
			Comment: "system",
		}

		for idx, lc := range kolConfig {
			isLastLevel := idx == len(kolConfig)-1

			if eligibleCount >= int64(lc.DeviceThreshold) && !isLastLevel {
				continue
			}

			kol.Level = lc.Level
			break
		}

		if setLevel, existing := adminAddedKols[userId]; existing && kol.Level < setLevel {
			kol.Level = setLevel
		}

		out[userId] = kol.Level
		updateKols = append(updateKols, kol)
	}

	if len(updateKols) == 0 {
		return out, nil
	}

	err = dao.UpsertKOLs(ctx, updateKols)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func updateUserL1Rewards(ctx context.Context, userRewards []*model.UserReward) error {
	var todos []*model.UserL1Reward

	for _, u := range userRewards {
		if int64(u.L1Reward) <= 0 {
			continue
		}

		todos = append(todos, &model.UserL1Reward{
			UserId: u.UserId,
			Reward: u.L1Reward,
		})

	}

	if err := dao.BulkUpdateUserL1Reward(ctx, todos); err != nil {
		return errs.Wrap(err, "BulkUpdateUserL1Reward")
	}

	return nil
}

func SumAllNodes() error {
	log.Info("start to sum all nodes")
	start := time.Now()
	defer func() {
		log.Infof("sum all nodes done, cost: %v", time.Since(start))
	}()

	ctx := context.Background()
	fullNodeInfo, err := dao.SumFullNodeInfoFromDeviceInfo(ctx)
	if err != nil {
		log.Errorf("count full node: %v", err)
		return err
	}

	assetCount, err := dao.CountAssets(ctx)
	if err != nil {
		log.Errorf("count assets: %v", err)
	}

	if assetCount == 0 {
		assetCount = 1
	}

	fullNodeInfo.TAverageReplica = formatter.ToFixed(float64(fullNodeInfo.TUpstreamFileCount)/float64(assetCount), 2)
	fullNodeInfo.Time = time.Now()
	fullNodeInfo.CreatedAt = time.Now()

	sum, err := dao.SumFilStorage(ctx)
	if err != nil {
		log.Errorf("CountStorageStats: %v", err)
	}
	fullNodeInfo.FBackupsFromTitan = float64(sum)

	err = dao.CacheFullNodeInfo(ctx, fullNodeInfo)
	if err != nil {
		log.Errorf("cache full node info: %v", err)
		return err
	}

	fTime := fullNodeInfo.Time
	fullNodeInfo.Time = time.Date(fTime.Year(), fTime.Month(), fTime.Day(), 0, 0, 0, 0, time.Local)

	if err = dao.UpsertFullNodeInfo(ctx, fullNodeInfo); err != nil {
		log.Errorf("upsert full node: %v", err)
		return err
	}

	return nil
}
