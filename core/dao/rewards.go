package dao

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	logging "github.com/ipfs/go-log/v2"
	"time"
)

var (
	tableNameDeviceInfoHour  = "device_info_hour"
	tableNameDeviceInfoDaily = "device_info_daily"
	log                      = logging.Logger("statistics")
)

func BulkUpsertDeviceInfoHours(ctx context.Context, hourInfos []*model.DeviceInfoHour) error {
	upsertStatement := fmt.Sprintf(`INSERT INTO %s (created_at, updated_at, hour_income, user_id, device_id,
				online_time, pkg_loss_ratio, latency, nat_ratio, disk_usage, upstream_traffic, downstream_traffic, retrieval_count, block_count, time)
			VALUES (:created_at, :updated_at, :hour_income, :user_id, :device_id, :online_time, :pkg_loss_ratio, :latency, 
			    :nat_ratio, :disk_usage, :upstream_traffic, :downstream_traffic, :retrieval_count, :block_count, :time) 
			 ON DUPLICATE KEY UPDATE updated_at = now(), hour_income = VALUES(hour_income),
			online_time = VALUES(online_time), pkg_loss_ratio = VALUES(pkg_loss_ratio), latency = VALUES(latency),
			upstream_traffic = VALUES(upstream_traffic), downstream_traffic = VALUES(downstream_traffic), retrieval_count = VALUES(retrieval_count), block_count = VALUES(block_count),
			nat_ratio = VALUES(nat_ratio), disk_usage = VALUES(disk_usage)`, tableNameDeviceInfoHour)
	_, err := DB.NamedExecContext(ctx, upsertStatement, hourInfos)
	return err
}

func BulkUpsertDeviceInfoDaily(ctx context.Context, dailyInfos []*model.DeviceInfoDaily) error {
	upsertStatement := fmt.Sprintf(`INSERT INTO %s (created_at, updated_at, income, user_id, device_id,
				online_time, pkg_loss_ratio, latency, nat_ratio, disk_usage, upstream_traffic, downstream_traffic, retrieval_count, block_count, time)
			VALUES (:created_at, :updated_at, :income, :user_id, :device_id,
				:online_time, :pkg_loss_ratio, :latency, :nat_ratio, :disk_usage, :upstream_traffic, :downstream_traffic, :retrieval_count, :block_count, :time) 
			 ON DUPLICATE KEY UPDATE updated_at = now(), income = VALUES(income),
			online_time = VALUES(online_time), pkg_loss_ratio = VALUES(pkg_loss_ratio), latency = VALUES(latency),
			upstream_traffic = VALUES(upstream_traffic), downstream_traffic = VALUES(downstream_traffic), retrieval_count = VALUES(retrieval_count), block_count = VALUES(block_count),
			nat_ratio = VALUES(nat_ratio), disk_usage = VALUES(disk_usage)`, tableNameDeviceInfoDaily)

	_, err := DB.NamedExecContext(ctx, upsertStatement, dailyInfos)
	return err
}

type DeviceStatistics struct {
	Date              string  `json:"date" db:"date"`
	NatRatio          float64 `json:"nat_ratio" db:"nat_ratio"`
	DiskUsage         float64 `json:"disk_usage" db:"disk_usage"`
	Latency           float64 `json:"latency" db:"latency"`
	PkgLossRatio      float64 `json:"pkg_loss_ratio" db:"pkg_loss_ratio"`
	Income            float64 `json:"income" db:"income"`
	OnlineTime        float64 `json:"online_time" db:"online_time"`
	UpstreamTraffic   float64 `json:"upstream_traffic" db:"upstream_traffic"`
	DownstreamTraffic float64 `json:"downstream_traffic" db:"downstream_traffic"`
	RetrievalCount    float64 `json:"retrieval_count" db:"retrieval_count"`
}

func GetDeviceInfoDailyHourList(ctx context.Context, cond *model.DeviceInfoHour, option QueryOption) ([]*DeviceStatistics, error) {
	sqlClause := fmt.Sprintf(`select date_format(time, '%%Y-%%m-%%d %%H') as date, avg(nat_ratio) as nat_ratio, 
	avg(disk_usage) as disk_usage, avg(latency) as latency, avg(pkg_loss_ratio) as pkg_loss_ratio, 
	min(online_time) as online_time,
	min(hour_income) as income,
	min(upstream_traffic) as upstream_traffic, 
	min(downstream_traffic) as downstream_traffic,
	min(retrieval_count) as retrieval_count
	from %s where device_id='%s' and time>='%s' and time<='%s' group by date order by date`, tableNameDeviceInfoHour, cond.DeviceID, option.StartTime, option.EndTime)
	var out []*DeviceStatistics
	err := DB.SelectContext(ctx, &out, sqlClause)
	if err != nil {
		return nil, err
	}

	var maxOne DeviceStatistics
	query := fmt.Sprintf(`SELECT 
		max(online_time) as online_time,
		max(hour_income) as income,
		max(upstream_traffic) as upstream_traffic, 
		max(downstream_traffic) as downstream_traffic,
		max(retrieval_count) as retrieval_count 
		FROM %s WHERE device_id='%s' AND time>='%s' AND time<='%s'`,
		tableNameDeviceInfoHour, cond.DeviceID, option.StartTime, option.EndTime)
	err = DB.GetContext(ctx, &maxOne, query)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return out, nil
	}

	end := len(out) - 1
	out[end].OnlineTime = maxOne.OnlineTime - out[end].OnlineTime
	out[end].Income = maxOne.Income - out[end].Income
	out[end].UpstreamTraffic = maxOne.UpstreamTraffic - out[end].UpstreamTraffic
	out[end].DownstreamTraffic = maxOne.DownstreamTraffic - out[end].DownstreamTraffic
	out[end].RetrievalCount = maxOne.RetrievalCount - out[end].RetrievalCount

	prevOne := *out[0]
	for i := 1; i < end; i++ {
		out[i].OnlineTime, prevOne.OnlineTime = out[i].OnlineTime-prevOne.OnlineTime, out[i].OnlineTime
		out[i].Income, prevOne.Income = out[i].Income-prevOne.Income, out[i].Income
		out[i].UpstreamTraffic, prevOne.UpstreamTraffic = out[i].UpstreamTraffic-prevOne.UpstreamTraffic, out[i].UpstreamTraffic
		out[i].DownstreamTraffic, prevOne.DownstreamTraffic = out[i].DownstreamTraffic-prevOne.DownstreamTraffic, out[i].DownstreamTraffic
		out[i].RetrievalCount, prevOne.RetrievalCount = out[i].RetrievalCount-prevOne.RetrievalCount, out[i].RetrievalCount
	}

	return handleHourList(out[1:]), err
}

func GetDeviceInfoDailyList(ctx context.Context, cond *model.DeviceInfoDaily, option QueryOption) ([]*DeviceStatistics, error) {
	var args []interface{}
	where := `WHERE 1=1`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}
	if option.StartTime != "" {
		where += ` AND time >= ?`
		args = append(args, option.StartTime)
	}
	if option.EndTime != "" {
		where += ` AND time <= ?`
		args = append(args, option.EndTime)
	}

	var out []*DeviceStatistics
	err := DB.SelectContext(ctx, &out, fmt.Sprintf(
		`SELECT DATE_FORMAT(time, '%%Y-%%m-%%d') as date, nat_ratio, disk_usage, latency, pkg_loss_ratio, income, online_time, upstream_traffic, 
    	downstream_traffic, retrieval_count FROM %s %s`, tableNameDeviceInfoDaily, where), args...)
	if err != nil {
		return nil, err
	}

	return handleDailyList(option.StartTime, option.EndTime, out), err
}

func handleDailyList(start, end string, in []*DeviceStatistics) []*DeviceStatistics {
	startTime, _ := time.Parse(utils.TimeFormatYMD, start)
	endTime, _ := time.Parse(utils.TimeFormatYMD, end)
	var oneDay = 24 * time.Hour
	dataKye := make(map[string]*DeviceStatistics)
	var out []*DeviceStatistics
	for _, data := range in {
		dataKye[data.Date] = data
	}
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(utils.TimeFormatYMD)
		startTime = startTime.Add(oneDay)
		val, ok := dataKye[key]
		var dataL DeviceStatistics
		if !ok {
			dataL.Date = startTime.Format(utils.TimeFormatMD)
			out = append(out, &dataL)
			continue
		}
		val.Date = startTime.Format(utils.TimeFormatMD)
		out = append(out, val)
	}

	return out

}

func handleHourList(in []*DeviceStatistics) []*DeviceStatistics {
	var oneHour = time.Hour
	now := time.Now()
	startTime := now.Add(-23 * oneHour)
	endTime := now
	dataKye := make(map[string]*DeviceStatistics)
	var out []*DeviceStatistics
	for _, data := range in {
		dataKye[data.Date] = data
	}
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(utils.TimeFormatYMDH)
		val, ok := dataKye[key]
		var dataL DeviceStatistics
		if !ok {
			dataL.Date = startTime.Format(utils.TimeFormatH) + ":00"
			out = append(out, &dataL)
			startTime = startTime.Add(oneHour)
			continue
		}
		val.Date = startTime.Format(utils.TimeFormatH) + ":00"
		out = append(out, val)
		startTime = startTime.Add(oneHour)
	}

	return out
}

func GetUserIncome(ctx context.Context, cond *model.DeviceInfo, option QueryOption) (map[string]map[string]interface{}, error) {
	sqlClause := fmt.Sprintf(`
		select date_format(b.time, '%%Y-%%m-%%d') as date, sum(b.income) as income  from %s a LEFT JOIN %s b on a.device_id = b.device_id 
    	and a.user_id = '%s' and date_format(b.time, '%%Y-%%m-%%d') >='%s' and date_format(b.time, '%%Y-%%m-%%d') <='%s' group by date`,
		tableNameDeviceInfo, tableNameDeviceInfoDaily, cond.UserID, option.StartTime, option.EndTime)
	dataS, err := GetQueryDataList(sqlClause)
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]interface{})
	for _, data := range dataS {
		_, ok := out[data["date"]]
		if !ok {
			out[data["date"]] = make(map[string]interface{})
		}
		out[data["date"]]["income"] = utils.StrToFloat(data["income"])
	}
	return out, nil
}

func GetDeviceInfoDailyByPage(ctx context.Context, cond *model.DeviceInfoDaily, option QueryOption) ([]*model.DeviceInfoDaily, int64, error) {
	var args []interface{}
	where := `WHERE 1=1`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}

	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 50
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	var total int64
	var out []*model.DeviceInfoDaily

	err := DB.GetContext(ctx, &total, fmt.Sprintf(
		`SELECT count(*) FROM %s %s`, tableNameDeviceInfo, where,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	err = DB.SelectContext(ctx, &out, fmt.Sprintf(
		`SELECT * FROM %s %s LIMIT %d OFFSET %s`, tableNameDeviceInfoDaily, where, limit, offset), args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

func GetRetrievalEventsFromDeviceByPage(ctx context.Context, cond *model.DeviceInfoHour, option QueryOption) ([]*model.DeviceInfoHour, int64, error) {
	var args []interface{}
	where := `WHERE 1=1`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 50
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	var total int64
	var out []*model.DeviceInfoHour

	err := DB.GetContext(ctx, &total, fmt.Sprintf(
		`select count(*)  from (
	select device_id, retrieval_count , upstream_traffic , created_at, 
	@a.retrieval_count AS pre_retrieval_count,
	@a.upstream_traffic AS pre_upstream_traffic,
	@a.retrieval_count := a.retrieval_count, 
	@a.upstream_traffic := a.upstream_traffic  
	from %s a ,
	(SELECT @a.retrieval_count := 0, @a.upstream_traffic := 0 ) b %s
) c where (c.retrieval_count - c.pre_retrieval_count) > 0`, tableNameDeviceInfoHour, where,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
select device_id, created_at, (c.retrieval_count - c.pre_retrieval_count) as retrieval_count, (c.upstream_traffic - c.pre_upstream_traffic) as upstream_traffic  
from (
	select device_id, retrieval_count , upstream_traffic , created_at, 
	@a.retrieval_count AS pre_retrieval_count,
	@a.upstream_traffic AS pre_upstream_traffic,
	@a.retrieval_count := a.retrieval_count, 
	@a.upstream_traffic := a.upstream_traffic  
	from %s a ,
	(SELECT @a.retrieval_count := 0, @a.upstream_traffic := 0 ) b %s
) c where (c.retrieval_count - c.pre_retrieval_count) > 0 ORDER BY created_at DESC limit %d offset %d`, tableNameDeviceInfoHour, where, limit, offset)
	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}
