package dao

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	logging "github.com/ipfs/go-log/v2"
	"strings"
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
			 ON DUPLICATE KEY UPDATE updated_at = now(), hour_income = :hour_income,
			online_time = :online_time, pkg_loss_ratio = :pkg_loss_ratio, latency = :latency,
			upstream_traffic = :upstream_traffic, downstream_traffic = :downstream_traffic, retrieval_count = :retrieval_count, block_count = :block_count,
			nat_ratio = :nat_ratio, disk_usage = :disk_usage`, tableNameDeviceInfoHour)
	tx := DB.MustBegin()
	defer tx.Rollback()

	for _, hourInfo := range hourInfos {
		_, err := tx.NamedExecContext(ctx, upsertStatement, hourInfo)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func BulkUpsertDeviceInfoDaily(ctx context.Context, dailyInfos []*model.DeviceInfoDaily) error {
	upsertStatement := fmt.Sprintf(`INSERT INTO %s (created_at, updated_at, income, user_id, device_id,
				online_time, pkg_loss_ratio, latency, nat_ratio, disk_usage, upstream_traffic, downstream_traffic, retrieval_count, block_count, time)
			VALUES (:created_at, :updated_at, :income, :user_id, :device_id,
				:online_time, :pkg_loss_ratio, :latency, :nat_ratio, :disk_usage, :upstream_traffic, :downstream_traffic, :retrieval_count, :block_count, :time) 
			 ON DUPLICATE KEY UPDATE updated_at = now(), income = :income,
			online_time = :online_time, pkg_loss_ratio = :pkg_loss_ratio, latency = :latency,
			upstream_traffic = :upstream_traffic, downstream_traffic = :downstream_traffic, retrieval_count = :retrieval_count, block_count = :block_count,
			nat_ratio = :nat_ratio, disk_usage = :disk_usage`, tableNameDeviceInfoDaily)
	tx := DB.MustBegin()
	defer tx.Rollback()

	for _, dailyInfo := range dailyInfos {
		_, err := tx.NamedExecContext(ctx, upsertStatement, dailyInfo)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

type DeviceStatistics struct {
	// 日期
	Date string `json:"date" db:"date"`
	// NAT率
	NatRatio float64 `json:"nat_ratio" db:"nat_ratio"`
	// 磁盘使用率
	DiskUsage float64 `json:"disk_usage" db:"disk_usage"`
	// 时延
	Latency float64 `json:"latency" db:"latency"`
	// 丢包率
	PkgLossRatio float64 `json:"pkg_loss_ratio" db:"pkg_loss_ratio"`
	// 收益
	Income float64 `json:"income" db:"income"`
	// 在线时长
	OnlineTime float64 `json:"online_time" db:"online_time"`
	// 上行流量
	UpstreamTraffic float64 `json:"upstream_traffic" db:"upstream_traffic"`
	// 下行流量
	DownstreamTraffic float64 `json:"downstream_traffic" db:"downstream_traffic"`
	// 检索数量
	RetrievalCount float64 `json:"retrieval_count" db:"retrieval_count"`
}

func GetDeviceInfoDailyHourList(ctx context.Context, cond *model.DeviceInfoHour, option QueryOption) ([]*DeviceStatistics, error) {
	sqlClause := fmt.Sprintf(`select date_format(time, '%%Y-%%m-%%d %%H') as date, avg(nat_ratio) as nat_ratio, 
	avg(disk_usage) as disk_usage, avg(latency) as latency, avg(pkg_loss_ratio) as pkg_loss_ratio, 
	max(hour_income) - min(hour_income) as income,
	max(online_time) - min(online_time) as online_time,
	max(upstream_traffic) - min(upstream_traffic) as upstream_traffic,
	max(downstream_traffic) - min(downstream_traffic) as downstream_traffic,
	max(retrieval_count) - min(retrieval_count) as retrieval_count
	from device_info_hour where device_id='%s' and time>='%s' and time<='%s' group by date order by id desc`, cond.DeviceID, option.StartTime, option.EndTime)
	var out []*DeviceStatistics
	err := DB.SelectContext(ctx, &out, sqlClause)
	if err != nil {
		return nil, err
	}
	var outNew []*DeviceStatistics
	var lasts string
	var num int
	for i, data := range out {
		nowTimeHour := strings.Split(data.Date, " ")[1]
		nowHour := nowTimeHour
	loop:
		if nowHour != lasts && lasts != "" {
			var dataL DeviceStatistics
			dataL.Date = lasts + ":00"
			outNew = append(outNew, &dataL)
			num += 1
			if lasts == "00" {
				lasts = "23"
			} else {
				lasts = fmt.Sprintf("%02d", utils.Str2Int(lasts)-1)
			}
			goto loop
		}
		if nowTimeHour == lasts || lasts == "" {
			data.Date = nowTimeHour + ":00"
			outNew = append(outNew, data)
			num += 1
			if nowTimeHour == "00" {
				lasts = "23"
			} else {
				lasts = fmt.Sprintf("%02d", utils.Str2Int(nowTimeHour)-1)
			}
		}
		if len(out) == i+1 && num <= 23 {
			if nowHour == "00" {
				nowHour = "23"
			} else {
				nowHour = fmt.Sprintf("%02d", utils.Str2Int(nowHour)-1)
			}
			goto loop
		}
		if num == 24 {
			break
		}
	}
	return reverseList(outNew), err
}

func reverseList(s []*DeviceStatistics) []*DeviceStatistics {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
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
		`SELECT DATE_FORMAT(time, '%%m-%%d') as date, nat_ratio, disk_usage, latency, pkg_loss_ratio, income, online_time, upstream_traffic, 
    	downstream_traffic, retrieval_count FROM %s %s`, tableNameDeviceInfoDaily, where), args...)
	if err != nil {
		return nil, err
	}

	return out, err
}

func GetUserIncome(ctx context.Context, cond *model.DeviceInfo, option QueryOption) (map[string]map[string]interface{}, error) {
	sqlClause := fmt.Sprintf("select date_format(time, '%%Y-%%m-%%d') as date, sum(income) as income from device_info_daily "+
		"where user_id='%s' and time>='%s' and time<='%s' group by date", cond.UserID, option.StartTime, option.EndTime)
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
