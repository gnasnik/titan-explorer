package dao

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/golang-module/carbon/v2"
	logging "github.com/ipfs/go-log/v2"
	"time"
)

var (
	tableNameDeviceInfoHour  = "device_info_hour"
	tableNameDeviceInfoDaily = "device_info_daily"
	log                      = logging.Logger("device_info")
)

func BulkUpsertDeviceInfoHours(ctx context.Context, hourInfos []*model.DeviceInfoHour) error {
	upsertStatement := fmt.Sprintf(`INSERT INTO %s (created_at, updated_at, hour_income, user_id, device_id,
				online_time, pkg_loss_ratio, latency, nat_ratio, disk_usage,disk_space,bandwidth_up,bandwidth_down, upstream_traffic, downstream_traffic, retrieval_count, block_count, time)
			VALUES (:created_at, :updated_at, :hour_income, :user_id, :device_id, :online_time, :pkg_loss_ratio, :latency, 
			    :nat_ratio, :disk_usage, :disk_space,:bandwidth_up,:bandwidth_down, :upstream_traffic, :downstream_traffic, :retrieval_count, :block_count, :time) 
			 ON DUPLICATE KEY UPDATE updated_at = now(), hour_income = VALUES(hour_income), user_id = VALUES(user_id),
			online_time = VALUES(online_time), pkg_loss_ratio = VALUES(pkg_loss_ratio), latency = VALUES(latency),
			upstream_traffic = VALUES(upstream_traffic), downstream_traffic = VALUES(downstream_traffic), retrieval_count = VALUES(retrieval_count), block_count = VALUES(block_count),
			nat_ratio = VALUES(nat_ratio), disk_usage = VALUES(disk_usage),disk_space = VALUES(disk_space),bandwidth_up = VALUES(bandwidth_up),bandwidth_down = VALUES(bandwidth_down)`, tableNameDeviceInfoHour)
	_, err := DB.NamedExecContext(ctx, upsertStatement, hourInfos)
	return err
}

func BulkUpsertDeviceInfoDaily(ctx context.Context, dailyInfos []*model.DeviceInfoDaily) error {
	upsertStatement := fmt.Sprintf(`INSERT INTO %s (created_at, updated_at, income, user_id, device_id,
				online_time, pkg_loss_ratio, latency, nat_ratio, disk_usage, disk_space,bandwidth_up,bandwidth_down, upstream_traffic, downstream_traffic, retrieval_count, block_count, time)
			VALUES (:created_at, :updated_at, :income, :user_id, :device_id,
				:online_time, :pkg_loss_ratio, :latency, :nat_ratio, :disk_usage, :disk_space,:bandwidth_up,:bandwidth_down, :upstream_traffic, :downstream_traffic, :retrieval_count, :block_count, :time) 
			 ON DUPLICATE KEY UPDATE updated_at = now(), income = VALUES(income),user_id = VALUES(user_id),
			online_time = VALUES(online_time), pkg_loss_ratio = VALUES(pkg_loss_ratio), latency = VALUES(latency),
			upstream_traffic = VALUES(upstream_traffic), downstream_traffic = VALUES(downstream_traffic), retrieval_count = VALUES(retrieval_count), block_count = VALUES(block_count),
			nat_ratio = VALUES(nat_ratio), disk_usage = VALUES(disk_usage), disk_space = VALUES(disk_space),bandwidth_up = VALUES(bandwidth_up),bandwidth_down = VALUES(bandwidth_down)`, tableNameDeviceInfoDaily)

	_, err := DB.NamedExecContext(ctx, upsertStatement, dailyInfos)
	return err
}

type DeviceStatistics struct {
	Date              string  `json:"date" db:"date"`
	NatRatio          float64 `json:"nat_ratio" db:"nat_ratio"`
	DiskUsage         float64 `json:"disk_usage" db:"disk_usage"`
	DiskSpace         float64 `json:"disk_space" db:"disk_space"`
	Latency           float64 `json:"latency" db:"latency"`
	PkgLossRatio      float64 `json:"pkg_loss_ratio" db:"pkg_loss_ratio"`
	Income            float64 `json:"income" db:"income"`
	OnlineTime        float64 `json:"online_time" db:"online_time"`
	BandwidthUp       float64 `db:"bandwidth_up" json:"bandwidth_up"`
	BandwidthDown     float64 `db:"bandwidth_down" json:"bandwidth_down"`
	UpstreamTraffic   float64 `json:"upstream_traffic" db:"upstream_traffic"`
	DownstreamTraffic float64 `json:"downstream_traffic" db:"downstream_traffic"`
	RetrievalCount    float64 `json:"retrieval_count" db:"retrieval_count"`
	BlockCount        float64 `json:"block_count" db:"block_count"`
	NodeCount         float64 `json:"node_count" db:"node_count"`
}

func GetDeviceHourlyIncome(ctx context.Context, nodeId string, option QueryOption) ([]*DeviceStatistics, error) {
	if option.StartTime == "" {
		option.StartTime = carbon.Now().StartOfDay().String()
	}

	if option.EndTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	}

	query := fmt.Sprintf(`select date_format(time, '%%H') as date, max(hour_income) - min(hour_income) as income from %s 
            where device_id='%s' and time>='%s' and time<='%s' group by date order by date`, tableNameDeviceInfoHour, nodeId, option.StartTime, option.EndTime)

	var out []*DeviceStatistics
	err := DB.SelectContext(ctx, &out, query)
	if err != nil {
		return nil, err
	}

	if len(out) > 0 {
		append24HoursData(out)
	}

	return out, err
}

func GetDeviceInfoDailyHourList(ctx context.Context, cond *model.DeviceInfoHour, option QueryOption) ([]*DeviceStatistics, error) {
	sqlClause := fmt.Sprintf(`select date_format(time, '%%Y-%%m-%%d %%H') as date, avg(nat_ratio) as nat_ratio, 
	avg(disk_usage) as disk_usage,avg(disk_space) as disk_space,avg(bandwidth_up) as bandwidth_up,avg(bandwidth_down) as bandwidth_down, avg(latency) as latency, avg(pkg_loss_ratio) as pkg_loss_ratio, 
	if( (max(online_time) - min(online_time)) > 60,60,(max(online_time) - min(online_time))) as online_time,
	max(hour_income) - min(hour_income) as income,
	max(upstream_traffic) - min(upstream_traffic) as upstream_traffic, 
	max(downstream_traffic) - min(downstream_traffic) as downstream_traffic,
	max(block_count) - min(block_count) as block_count,
	max(retrieval_count) - min(retrieval_count) as retrieval_count
	from %s where device_id='%s' and time>='%s' and time<='%s' group by date order by date`, tableNameDeviceInfoHour, cond.DeviceID, option.StartTime, option.EndTime)
	var out []*DeviceStatistics
	err := DB.SelectContext(ctx, &out, sqlClause)
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		return append24HoursData(out), err
	}
	return out, err
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

	var result []*DeviceStatistics
	err := DB.SelectContext(ctx, &result, fmt.Sprintf(
		`SELECT DATE_FORMAT(time, '%%Y-%%m-%%d') as date, nat_ratio, disk_usage,disk_space,bandwidth_up,bandwidth_down, latency, pkg_loss_ratio, income, online_time, upstream_traffic, 
    	downstream_traffic, retrieval_count,block_count FROM %s %s`, tableNameDeviceInfoDaily, where), args...)
	if err != nil {
		return nil, err
	}

	return handleDailyList(result), err
}

func GetNodesInfoDailyList(ctx context.Context, cond *model.DeviceInfoDaily, option QueryOption) ([]*DeviceStatistics, error) {
	var args []interface{}
	where := `WHERE 1=1`
	if cond.UserID != "" {
		where += ` AND user_id = ?`
		args = append(args, cond.UserID)
	}
	if option.StartTime != "" {
		where += ` AND time >= ?`
		args = append(args, option.StartTime)
	}
	if option.EndTime != "" {
		where += ` AND time <= ?`
		args = append(args, option.EndTime)
	}

	var result []*DeviceStatistics
	err := DB.SelectContext(ctx, &result, fmt.Sprintf(
		`SELECT DATE_FORMAT(time, '%%Y-%%m-%%d') as date, COUNT(DISTINCT(device_id)) as node_count,nat_ratio, ROUND(sum(disk_usage*disk_space/100),4) as disk_usage,ROUND(sum(disk_space),4) as disk_space,  ROUND(sum(income),2) as income, ROUND(sum(upstream_traffic),2) as upstream_traffic, 
    	ROUND(sum(bandwidth_up),2) as bandwidth_up,ROUND(sum(bandwidth_down),2) as bandwidth_down,
    	ROUND(sum(downstream_traffic),2) as downstream_traffic, ROUND(sum(retrieval_count),2) as retrieval_count,ROUND(sum(block_count),2) as block_count FROM %s %s group by date`, tableNameDeviceInfoDaily, where), args...)
	if err != nil {
		return nil, err
	}

	return handleDailyList(result), err
}

func handleDailyList(deviceStat []*DeviceStatistics) []*DeviceStatistics {
	now := time.Now()
	startTime, endTime := now, now
	oneDay := 24 * time.Hour
	deviceInDate := make(map[string]*DeviceStatistics)

	for _, data := range deviceStat {
		t, _ := time.Parse(time.DateOnly, data.Date)

		if t.Before(startTime) {
			startTime = t
		}

		if t.After(endTime) {
			endTime = t
		}

		deviceInDate[data.Date] = data
	}

	var out []*DeviceStatistics
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(formatter.TimeFormatDateOnly)
		val, ok := deviceInDate[key]
		if !ok {
			val = &DeviceStatistics{}
		}
		val.Date = startTime.Format(formatter.TimeFormatMD)
		out = append(out, val)
		startTime = startTime.Add(oneDay)
	}

	return out

}

func append24HoursData(in []*DeviceStatistics) []*DeviceStatistics {
	now := time.Now()
	oneHour := time.Hour
	startTime := now.Add(-23 * oneHour)
	endTime := now
	deviceInDate := make(map[string]*DeviceStatistics)
	var out []*DeviceStatistics
	for _, data := range in {
		deviceInDate[data.Date] = data
	}
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(formatter.TimeFormatYMDH)
		val, ok := deviceInDate[key]
		if !ok {
			val = &DeviceStatistics{}
		}
		val.Date = fmt.Sprintf("%d:00", startTime.Hour())
		out = append(out, val)
		startTime = startTime.Add(oneHour)
	}

	return out
}

func GetUserIncome(cond *model.DeviceInfo, option QueryOption) (map[string]map[string]interface{}, error) {
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
		out[data["date"]]["income"] = formatter.StrToFloat(data["income"])
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
		`SELECT * FROM %s %s LIMIT %d OFFSET %d`, tableNameDeviceInfoDaily, where, limit, offset), args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

func GetDeviceInfoByDeviceId(ctx context.Context, cond *model.DeviceInfoDaily, option QueryOption) ([]*model.DeviceInfoDaily, int64, error) {
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
		`SELECT * FROM %s %s LIMIT %d OFFSET %d`, tableNameDeviceInfo, where, limit, offset), args...)
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
