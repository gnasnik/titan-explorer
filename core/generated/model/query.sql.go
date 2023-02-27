// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.0
// source: query.sql

package model

import (
	"context"
)

const getDeviceInfo = `-- name: GetDeviceInfo :one
SELECT created_at, updated_at, deleted_at, bound_at, device_id, scheduler_id, node_type, device_rank, device_name, user_id, sn_code, operator, network_type, system_version, product_type, network_info, external_ip, internal_ip, ip_location, ip_country, ip_province, ip_city, latitude, longitude, mac_location, nat_type, upnp, pkg_loss_ratio, nat_ratio, latency, cpu_usage, cpu_cores, memory_usage, memory, disk_usage, disk_space, bind_status, work_status, device_status, active_status, disk_type, io_system, online_time, today_online_time, today_profit, yesterday_profit, seven_days_profit, month_profit, cumulative_profit, bandwidth_up, bandwidth_down, total_download, total_upload, block_count, download_count FROM ` + "`" + `device_info` + "`" + ` WHERE device_id = ? LIMIT 1
`

func (q *Queries) GetDeviceInfo(ctx context.Context, db DBTX, deviceID string) (DeviceInfo, error) {
	row := db.QueryRowContext(ctx, getDeviceInfo, deviceID)
	var i DeviceInfo
	err := row.Scan(
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.BoundAt,
		&i.DeviceID,
		&i.SchedulerID,
		&i.NodeType,
		&i.DeviceRank,
		&i.DeviceName,
		&i.UserID,
		&i.SnCode,
		&i.Operator,
		&i.NetworkType,
		&i.SystemVersion,
		&i.ProductType,
		&i.NetworkInfo,
		&i.ExternalIp,
		&i.InternalIp,
		&i.IpLocation,
		&i.IpCountry,
		&i.IpProvince,
		&i.IpCity,
		&i.Latitude,
		&i.Longitude,
		&i.MacLocation,
		&i.NatType,
		&i.Upnp,
		&i.PkgLossRatio,
		&i.NatRatio,
		&i.Latency,
		&i.CpuUsage,
		&i.CpuCores,
		&i.MemoryUsage,
		&i.Memory,
		&i.DiskUsage,
		&i.DiskSpace,
		&i.BindStatus,
		&i.WorkStatus,
		&i.DeviceStatus,
		&i.ActiveStatus,
		&i.DiskType,
		&i.IoSystem,
		&i.OnlineTime,
		&i.TodayOnlineTime,
		&i.TodayProfit,
		&i.YesterdayProfit,
		&i.SevenDaysProfit,
		&i.MonthProfit,
		&i.CumulativeProfit,
		&i.BandwidthUp,
		&i.BandwidthDown,
		&i.TotalDownload,
		&i.TotalUpload,
		&i.BlockCount,
		&i.DownloadCount,
	)
	return i, err
}
