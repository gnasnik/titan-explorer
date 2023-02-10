package statistics

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	"github.com/golang-module/carbon/v2"
	"time"
)

func addDeviceInfoHours(ctx context.Context, deviceInfo []*model.DeviceInfo) error {
	log.Info("start to fetch device info hours")
	start := time.Now()
	defer func() {
		log.Infof("fetch device info hours done, cost: %v", time.Since(start))
	}()

	var upsertDevice []*model.DeviceInfoHour
	for _, device := range deviceInfo {
		var deviceInfoHour model.DeviceInfoHour
		deviceInfoHour.Time = start
		deviceInfoHour.DiskUsage = device.DiskUsage
		deviceInfoHour.DeviceID = device.DeviceID
		deviceInfoHour.PkgLossRatio = device.PkgLossRatio
		deviceInfoHour.HourIncome = device.CumulativeProfit
		deviceInfoHour.OnlineTime = device.OnlineTime
		deviceInfoHour.Latency = device.Latency
		deviceInfoHour.DiskUsage = device.DiskUsage
		deviceInfoHour.UpstreamTraffic = device.TotalUpload
		deviceInfoHour.DownstreamTraffic = device.TotalDownload
		deviceInfoHour.RetrievalCount = device.DownloadCount
		deviceInfoHour.BlockCount = device.BlockCount
		deviceInfoHour.CreatedAt = time.Now()
		deviceInfoHour.UpdatedAt = time.Now()
		upsertDevice = append(upsertDevice, &deviceInfoHour)
	}

	err := dao.BulkUpsertDeviceInfoHours(ctx, upsertDevice)
	if err != nil {
		log.Errorf("bulk upsert device info: %v", err)
	}
	return nil
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

	startOfTodayTime := carbon.Now().StartOfDay().String()
	endOfTodayTime := carbon.Now().Tomorrow().StartOfDay().String()
	sqlClause := fmt.Sprintf(`select user_id, device_id, date_format(time, '%%Y-%%m-%%d') as date, 
			avg(nat_ratio) as nat_ratio, avg(disk_usage) as disk_usage, avg(latency) as latency, 
			avg(pkg_loss_ratio) as pkg_loss_ratio, 
			max(hour_income) - min(hour_income) as hour_income,
			max(online_time) - min(online_time) as online_time,
			max(upstream_traffic) - min(upstream_traffic) as upstream_traffic,
			max(downstream_traffic) - min(downstream_traffic) as downstream_traffic,
			max(retrieval_count) - min(retrieval_count) as retrieval_count,
			max(block_count) - min(block_count) as block_count  from device_info_hour                                                                                      
			where time>='%s' and time<='%s' group by device_id`, startOfTodayTime, endOfTodayTime)
	datas, err := dao.GetQueryDataList(sqlClause)
	if err != nil {
		return err
	}

	var dailyInfos []*model.DeviceInfoDaily
	for _, data := range datas {
		var daily model.DeviceInfoDaily
		daily.Time, _ = time.Parse(utils.TimeFormatYMD, data["date"])
		daily.DiskUsage = utils.Str2Float64(data["disk_usage"])
		daily.NatRatio = utils.Str2Float64(data["nat_ratio"])
		daily.Income = utils.Str2Float64(data["hour_income"])
		daily.OnlineTime = utils.Str2Float64(data["online_time"])
		daily.UpstreamTraffic = utils.Str2Float64(data["upstream_traffic"])
		daily.DownstreamTraffic = utils.Str2Float64(data["downstream_traffic"])
		daily.RetrievalCount = utils.Str2Int64(data["retrieval_count"])
		daily.BlockCount = utils.Str2Int64(data["block_count"])
		daily.PkgLossRatio = utils.Str2Float64(data["pkg_loss_ratio"])
		daily.Latency = utils.Str2Float64(data["latency"])
		daily.DeviceID = data["device_id"]
		daily.UserID = data["user_id"]
		daily.CreatedAt = time.Now()
		daily.UpdatedAt = time.Now()
		dailyInfos = append(dailyInfos, &daily)
	}

	err = dao.BulkUpsertDeviceInfoDaily(context.Background(), dailyInfos)
	if err != nil {
		return err
	}

	return nil
}

func (s *Statistic) SumDeviceInfoProfit() error {
	log.Info("start to sum device info profit")
	start := time.Now()
	defer func() {
		log.Infof("sum device info profit done, cost: %v", time.Since(start))
	}()

	if err := s.SumDeviceInfoDaily(); err != nil {
		log.Errorf("sum device info daily: %v", err)
		return err
	}

	updatedDevices := make(map[string]*model.DeviceInfo)
	startOfTodayTime := carbon.Now().StartOfDay().String()
	endOfTodayTime := carbon.Now().EndOfDay().String()
	startOfYesterday := carbon.Yesterday().StartOfDay().String()
	endOfYesterday := carbon.Yesterday().EndOfDay().String()
	dataY := QueryDataByDate(startOfYesterday, endOfYesterday)
	for _, data := range dataY {
		_, ok := updatedDevices[data["device_id"]]
		if !ok {
			updatedDevices[data["device_id"]] = &model.DeviceInfo{
				DeviceID: data["device_id"],
			}
		}
		updatedDevices[data["device_id"]].YesterdayProfit = utils.Str2Float64(data["income"])
	}

	startOfWeekTime := carbon.Now().SubDays(6).StartOfDay().String()
	dataS := QueryDataByDate(startOfWeekTime, endOfTodayTime)

	for _, data := range dataS {
		_, ok := updatedDevices[data["device_id"]]
		if !ok {
			updatedDevices[data["device_id"]] = &model.DeviceInfo{
				DeviceID: data["device_id"],
			}
		}
		updatedDevices[data["device_id"]].SevenDaysProfit = utils.Str2Float64(data["income"])
	}

	startOfMonthTime := carbon.Now().SubDays(29).StartOfDay().String()
	dataM := QueryDataByDate(startOfMonthTime, endOfTodayTime)

	for _, data := range dataM {
		_, ok := updatedDevices[data["device_id"]]
		if !ok {
			updatedDevices[data["device_id"]] = &model.DeviceInfo{
				DeviceID: data["device_id"],
			}
		}
		updatedDevices[data["device_id"]].MonthProfit = utils.Str2Float64(data["income"])
	}

	dataT := QueryDataByDate(startOfTodayTime, endOfTodayTime)
	for _, data := range dataT {
		_, ok := updatedDevices[data["device_id"]]
		if !ok {
			updatedDevices[data["device_id"]] = &model.DeviceInfo{
				DeviceID: data["device_id"],
			}
		}
		updatedDevices[data["device_id"]].TodayProfit = utils.Str2Float64(data["income"])
		updatedDevices[data["device_id"]].TodayOnlineTime = utils.Str2Float64(data["online_time"])
	}

	var deviceInfos []*model.DeviceInfo
	for _, deviceInfo := range updatedDevices {
		deviceInfos = append(deviceInfos, deviceInfo)
	}

	if err := dao.BulkUpdateDeviceInfo(context.Background(), deviceInfos); err != nil {
		log.Errorf("bulk update devices: %v", err)
	}

	return nil
}

func (s *Statistic) SumAllNodes() error {
	log.Info("start to sum all nodes")
	start := time.Now()
	defer func() {
		log.Infof("sum all nodes done, cost: %v", time.Since(start))
	}()
	fullNodeInfo, err := dao.SumFullNodeInfoFromDeviceInfo(s.ctx)
	if err != nil {
		log.Errorf("count full node: %v", err)
		return err
	}

	systemInfo, err := dao.SumSystemInfo(s.ctx)
	if err != nil {
		log.Errorf("sum system info: %v", err)
		return err
	}

	fullNodeInfo.TotalCarfile = systemInfo.CarFileCount
	fullNodeInfo.RetrievalCount = systemInfo.DownloadCount
	fullNodeInfo.NextElectionTime = time.Unix(systemInfo.NextElectionTime, 0)

	fullNodeInfo.Time = time.Now()
	fullNodeInfo.CreatedAt = time.Now()
	err = dao.CacheFullNodeInfo(s.ctx, fullNodeInfo)
	if err != nil {
		log.Errorf("cache full node info: %v", err)
		return err
	}
	return nil
}

func (s *Statistic) UpdateDeviceRank() error {
	log.Info("start to rank device info")
	start := time.Now()
	defer func() {
		log.Infof("rank device info done, cost: %v", time.Since(start))
	}()
	if err := dao.RankDeviceInfo(s.ctx); err != nil {
		log.Errorf("ranking device info: %v", err)
		return err
	}
	return nil
}
