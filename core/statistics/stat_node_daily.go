package statistics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/golang-module/carbon/v2"
	errs "github.com/pkg/errors"
	"time"
)

// addDeviceInfoHours 写入 device_info_hour 表
func addDeviceInfoHours(ctx context.Context, upsertDevice []*model.DeviceInfoHour) error {
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

func getDeviceUserId(ctx context.Context, deviceId string) string {
	deviceOrdinaryInfo, err := dao.GetDeviceInfo(ctx, deviceId)
	if err != nil {
		log.Errorf("set device info: %v", err)
		return ""
	}

	if deviceOrdinaryInfo.UserID != "" {
		err = dao.SetDeviceUserIdToCache(ctx, deviceId, deviceOrdinaryInfo.UserID)
		if err != nil {
			log.Errorf("set device user to cahce: %v", err)
		}
	}

	return deviceOrdinaryInfo.UserID
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

func (n *NodeFetcher) SumDeviceInfoProfit() error {
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

func SumUserDeviceReward(ctx context.Context) error {
	log.Info("start to sum user device reward")
	start := time.Now()
	defer func() {
		log.Infof("sum sum user device reward done, cost: %v", time.Since(start))
	}()

	sumReward, err := dao.GetSumUserDeviceReward(ctx)
	if err != nil {
		return err
	}

	referralRewards, err := dao.SumUserReferralReward2(ctx)
	if err != nil {
		return err
	}

	var count int
	users := make([]*model.User, 0)
	for _, user := range sumReward {

		if rw, ok := referralRewards[user.Username]; ok {
			user.RefereralReward = rw
		}

		count++
		users = append(users, user)

		if len(users) == 1000 || count == len(sumReward) {
			if err := dao.BulkUpdateUserReward(context.Background(), users); err != nil {
				log.Errorf("bulk update devices: %v", err)
			}
			users = make([]*model.User, 0)
		}
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

	// todo NextElectionTime
	systemInfo, err := dao.SumSystemInfo(ctx)
	if err != nil {
		log.Errorf("sum system info: %v", err)
		return err
	}
	AssetCount := config.GNodesInfo.AssetCount
	if AssetCount == 0 {
		AssetCount = 1
	}
	fullNodeInfo.TAverageReplica = formatter.ToFixed(float64(fullNodeInfo.TUpstreamFileCount)/float64(AssetCount), 2)
	fullNodeInfo.TotalCarfile = systemInfo.CarFileCount
	fullNodeInfo.RetrievalCount = systemInfo.DownloadCount
	fullNodeInfo.NextElectionTime = systemInfo.NextElectionTime
	fullNodeInfo.Time = time.Now()
	fullNodeInfo.CreatedAt = time.Now()

	stats, err := dao.CountStorageStats(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Errorf("CountStorageStats: %v", err)
	}
	if stats != nil {
		fullNodeInfo.FBackupsFromTitan = stats.TotalSize
	} else {
		sum, err := dao.SumFilStorage(ctx)
		if err != nil {
			log.Errorf("CountStorageStats: %v", err)
		}
		fullNodeInfo.FBackupsFromTitan = float64(sum)
	}

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
