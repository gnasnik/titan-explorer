package dao

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
)

var tableNameDeviceInfo = "device_info"

func GetDeviceInfoList(ctx context.Context, cond *model.DeviceInfo, option QueryOption) ([]*model.DeviceInfo, int64, error) {
	var args []interface{}
	where := `WHERE device_id <> '' AND active_status = 1`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}
	if cond.UserID != "" {
		where += ` AND user_id = ?`
		args = append(args, cond.UserID)
	}
	if cond.DeviceStatus != "" && cond.DeviceStatus != "allDevices" {
		where += ` AND device_status = ?`
		args = append(args, cond.DeviceStatus)
	}
	if cond.IpLocation != "" && cond.IpLocation != "0" {
		where += ` AND ip_location = ?`
		args = append(args, cond.IpLocation)
	}
	if cond.NodeType > 0 {
		where += ` AND node_type = ?`
		args = append(args, cond.NodeType)
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
	var out []*model.DeviceInfo

	err := DB.GetContext(ctx, &total, fmt.Sprintf(
		`SELECT count(*) FROM %s %s`, tableNameDeviceInfo, where,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	err = DB.SelectContext(ctx, &out, fmt.Sprintf(
		`SELECT * FROM %s %s ORDER BY device_rank LIMIT %d OFFSET %d`, tableNameDeviceInfo, where, limit, offset,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

func HandleMapInfo(in []*model.DeviceInfo) []map[string]interface{} {
	type MapObject map[string]interface{}
	var mapInfoOut []map[string]interface{}
	for _, m := range in {
		mapInfoOut = append(mapInfoOut, MapObject{
			"name":  m.IpCity,
			"value": []float64{m.Latitude, m.Longitude},
		})

	}
	return mapInfoOut

}

func GetDeviceInfoByID(ctx context.Context, deviceID string) (*model.DeviceInfo, error) {
	var out model.DeviceInfo
	if err := DB.QueryRowxContext(ctx, fmt.Sprintf(
		`SELECT * FROM %s WHERE device_id = ?`, tableNameDeviceInfo), deviceID,
	).StructScan(&out); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func UpdateUserDeviceInfo(ctx context.Context, deviceInfo *model.DeviceInfo) error {
	_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET user_id = :user_id, updated_at = now(),bind_status = :bind_status WHERE device_id = :device_id`, tableNameDeviceInfo),
		deviceInfo)
	return err
}
func UpdateDeviceName(ctx context.Context, deviceInfo *model.DeviceInfo) error {
	_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET updated_at = now(),device_name = :device_name WHERE device_id = :device_id`, tableNameDeviceInfo),
		deviceInfo)
	return err
}
func BulkUpsertDeviceInfo(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	statement := upsertDeviceInfoStatement()
	_, err := DB.NamedExecContext(ctx, statement, deviceInfos)
	return err
}

func BulkUpdateDeviceInfo(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	for _, device := range deviceInfos {
		_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
			`UPDATE %s SET today_online_time = :today_online_time, today_profit = :today_profit,
				yesterday_profit = :yesterday_profit, seven_days_profit = :seven_days_profit, month_profit = :month_profit, 
				updated_at = now() WHERE device_id = :device_id`, tableNameDeviceInfo),
			device)
		if err != nil {
			return err
		}
	}
	return nil
}

func upsertDeviceInfoStatement() string {
	insertStatement := fmt.Sprintf(
		`INSERT INTO %s (device_id, node_type, device_name, user_id, sn_code, operator,
				network_type, system_version, product_type, active_status,
				network_info, external_ip, internal_ip, ip_location, ip_country, ip_province, ip_city, latitude, longitude, mac_location, nat_type, upnp,
				pkg_loss_ratio, nat_ratio, latency, cpu_usage, memory_usage, cpu_cores, memory, disk_usage, disk_space, work_status,
				device_status, disk_type, io_system, online_time, today_online_time, today_profit, total_upload, total_download, download_count, block_count,
				yesterday_profit, seven_days_profit, month_profit, cumulative_profit, bandwidth_up, bandwidth_down, created_at, updated_at)
			VALUES (:device_id, :node_type, :device_name, :user_id, :sn_code, :operator,
			    :network_type, :system_version, :product_type, :active_status,
			    :network_info, :external_ip, :internal_ip, :ip_location, :ip_country, :ip_province, :ip_city, :latitude, :longitude, :mac_location, :nat_type, :upnp, 
			    :pkg_loss_ratio, :nat_ratio, :latency, :cpu_usage, :memory_usage, :cpu_cores, :memory, :disk_usage, :disk_space, :work_status, 
			    :device_status, :disk_type, :io_system, :online_time, :today_online_time, :today_profit, :total_upload, :total_download, :download_count, block_count,
				:yesterday_profit, :seven_days_profit, :month_profit, :cumulative_profit, :bandwidth_up, :bandwidth_down, now(), now())`, tableNameDeviceInfo,
	)
	updateStatement := ` ON DUPLICATE KEY UPDATE node_type = VALUES(node_type),  device_name = VALUES(device_name),
				sn_code = VALUES(sn_code),  operator = VALUES(operator), network_type = VALUES(network_type), active_status = VALUES(active_status),
				system_version = VALUES(system_version),  product_type = VALUES(product_type), network_info = VALUES(network_info), cumulative_profit = VALUES(cumulative_profit),
				external_ip = VALUES(external_ip), internal_ip = VALUES(internal_ip), ip_location = VALUES(ip_location), ip_country = VALUES(ip_country), ip_province = VALUES(ip_province), ip_city = VALUES(ip_city), 
				latitude = VALUES(latitude), longitude = VALUES(longitude), mac_location = VALUES(mac_location),  nat_type = VALUES(nat_type),  upnp = VALUES(upnp), 
				pkg_loss_ratio = VALUES(pkg_loss_ratio), online_time = VALUES(online_time),
				nat_ratio = VALUES(nat_ratio), latency = VALUES(latency),  cpu_usage = VALUES(cpu_usage), cpu_cores = VALUES(cpu_cores),  memory_usage = VALUES(memory_usage), memory = VALUES(memory),
				disk_usage = VALUES(disk_usage), disk_space = VALUES(disk_space), work_status = VALUES(work_status), device_status = VALUES(device_status),  disk_type = VALUES(disk_type),
 				total_upload = VALUES(total_upload), total_download = VALUES(total_download), download_count = VALUES(download_count), block_count = VALUES(block_count),
				io_system = VALUES(io_system), bandwidth_up = VALUES(bandwidth_up), bandwidth_down = VALUES(bandwidth_down), updated_at = now()`
	return insertStatement + updateStatement
}

func GetAllAreaFromDeviceInfo(ctx context.Context) ([]string, error) {
	queryStatement := fmt.Sprintf(`SELECT ip_location FROM %s GROUP BY ip_location;`, tableNameDeviceInfo)
	var out []string
	err := DB.SelectContext(ctx, &out, queryStatement)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func SumFullNodeInfoFromDeviceInfo(ctx context.Context) (*model.FullNodeInfo, error) {
	queryStatement := fmt.Sprintf(`SELECT count( device_id ) AS total_node_count ,  SUM(IF(node_type = 1, 1, 0)) AS edge_count, 
       SUM(IF(node_type = 2, 1, 0)) AS candidate_count, SUM(IF(node_type = 3, 1, 0)) AS validator_count, ROUND(SUM( disk_space),4) AS total_storage, 
       ROUND(SUM(bandwidth_up),2) AS total_upstream_bandwidth, ROUND(SUM(bandwidth_down),2) AS total_downstream_bandwidth FROM %s where active_status = 1;`, tableNameDeviceInfo)

	var out model.FullNodeInfo
	if err := DB.QueryRowxContext(ctx, queryStatement).StructScan(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UserDeviceProfile struct {
	// 累计收益
	CumulativeProfit float64 `json:"cumulative_profit" db:"cumulative_profit" swaggertype:"number"`
	// 昨日收益
	YesterdayProfit float64 `json:"yesterday_profit" db:"yesterday_profit" swaggertype:"number"`
	// 今日收益
	TodayProfit float64 `json:"today_profit" db:"today_profit"  swaggertype:"number"`
	// 近7天收益
	SevenDaysProfit float64 `json:"seven_days_profit" db:"seven_days_profit"  swaggertype:"number"`
	// 近30天收益
	MonthProfit float64 `json:"month_profit" db:"month_profit"  swaggertype:"number"`
	// 设备总数
	TotalNum int64 `json:"total_num" db:"total_num"  swaggertype:"integer"`
	// 在线设备数
	OnlineNum int64 `json:"online_num" db:"online_num"  swaggertype:"integer"`
	// 设备离线
	OfflineNum int64 `json:"offline_num" db:"offline_num"  swaggertype:"integer"`
	// 设备异常
	AbnormalNum int64 `json:"abnormal_num" db:"abnormal_num"  swaggertype:"integer"`
	// 总上行速度
	TotalBandwidth float64 `json:"total_bandwidth" db:"total_bandwidth"  swaggertype:"number"`
}

func CountUserDeviceInfo(ctx context.Context, userID string) (*UserDeviceProfile, error) {
	queryStatement := fmt.Sprintf(`SELECT COALESCE(sum(cumulative_profit),0) as cumulative_profit, COALESCE(sum(yesterday_profit),0) as yesterday_profit, 
COALESCE(sum(today_profit),0) as today_profit, COALESCE(sum(seven_days_profit),0) as seven_days_profit, COALESCE(sum(month_profit),0) as month_profit, count(*) as total_num, 
count(IF(device_status = 'online', 1, NULL)) as online_num ,count(IF(device_status = 'offline', 1, NULL)) as offline_num, 
count(IF(device_status = 'abnormal', 1, NULL)) as abnormal_num, COALESCE(sum(bandwidth_up),0) as total_bandwidth from %s where user_id = ? and active_status = 1;`, tableNameDeviceInfo)

	var out UserDeviceProfile
	if err := DB.QueryRowxContext(ctx, queryStatement, userID).StructScan(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func RankDeviceInfo(ctx context.Context) error {
	tx := DB.MustBegin()
	defer tx.Rollback()
	tx.MustExec("SET @r=0;")
	queryStatement := fmt.Sprintf(`UPDATE %s SET device_rank= @r:= (@r+1) ORDER BY cumulative_profit DESC;`, tableNameDeviceInfo)
	_, err := tx.ExecContext(ctx, queryStatement)
	if err != nil {
		return err
	}
	return tx.Commit()
}
