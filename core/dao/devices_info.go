package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/go-redis/redis/v9"
	"github.com/jmoiron/sqlx"
)

var (
	tableNameDeviceInfo = "device_info"
)

type ActiveInfoOut struct {
	DeviceId     string `db:"device_id" json:"device_id"`
	ActiveStatus string `db:"active_status" json:"active_status"`
	Secret       string `db:"secret" json:"secret"`
}

type MapInfo struct {
	Name     string    `json:"name"`
	NodeType string    `json:"nodeType"`
	Ip       string    `json:"ip"`
	Value    []float64 `json:"value"`
}

func CacheMapInfo(ctx context.Context, mapInfo []*MapInfo, lang model.Language) error {
	key := fmt.Sprintf("TITAN::MAPINFO::%s", lang)

	data, err := json.Marshal(mapInfo)
	if err != nil {
		return err
	}

	expiration := time.Minute * 5
	_, err = RedisCache.Set(ctx, key, data, expiration).Result()
	if err != nil {
		log.Errorf("set chain head: %v", err)
	}

	return nil
}

func GetMapInfoFromCache(ctx context.Context, lang model.Language) ([]*MapInfo, error) {
	key := fmt.Sprintf("TITAN::MAPINFO::%s", lang)
	result, err := RedisCache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var out []*MapInfo
	err = json.Unmarshal([]byte(result), &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetDeviceMapInfo(ctx context.Context, lang model.Language, deviceId string) ([]*MapInfo, error) {
	location := "location_en"
	if lang == model.LanguageCN {
		location = "location_cn"
	}

	var where string
	if deviceId != "" {
		where = fmt.Sprintf(" and device_id = '%s'", deviceId)
	}

	query := fmt.Sprintf(`select t.name, CONCAT(
    SUBSTRING_INDEX(t.external_ip, '.', 1), 
    '.xxx.xxx.', 
    SUBSTRING_INDEX(t.external_ip, '.', -1)
  ) AS ip, t.node_type, t.longitude, t.latitude from  (select IF(lc.city <> '', lc.city, lc.country) as name, external_ip , d.node_type, d.longitude, d.latitude from device_info d  
      left join %s lc on d.external_ip = lc.ip  where device_status_code = 1 and ip_country <> 'China' %s) t group by t.external_ip`, location, where)

	rows, err := DB.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []*MapInfo

	for rows.Next() {
		var (
			name, nodeType, ip string
			lat, long          float64
		)

		if err := rows.Scan(&name, &ip, &nodeType, &long, &lat); err != nil {
			continue
		}

		if len(out) >= 50000 {
			break
		}

		out = append(out, &MapInfo{
			Name:     name,
			NodeType: nodeType,
			Ip:       ip,
			Value:    []float64{lat, long},
		})
	}

	return out, nil
}

func GetDeviceDistribution(ctx context.Context, lang model.Language) ([]*model.DeviceDistribution, error) {
	table := "location_en"
	if lang != "" {
		table = fmt.Sprintf("location_%s", lang)
	}
	query := fmt.Sprintf(`select IFNULL(l.country, '') as country, count(d.device_id) as count from device_info d left join %s l on d.external_ip = l.ip where device_status_code = 1 group by d.ip_country order by count desc limit 10;`, table)
	var out []*model.DeviceDistribution
	err := DB.SelectContext(ctx, &out, query)
	return out, err
}

func GetDeviceInfoList(ctx context.Context, cond *model.DeviceInfo, option QueryOption) ([]*model.DeviceInfo, int64, error) {
	var args []interface{}
	where := `WHERE device_id <> ''`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}
	if cond.BindStatus != "" {
		where += ` AND bind_status = ?`
		args = append(args, cond.BindStatus)
	}
	if cond.ActiveStatus < 10 {
		where += ` AND active_status = ?`
		args = append(args, cond.ActiveStatus)
	}
	if cond.UserID != "" || option.NotBound == "1" {
		where += ` AND user_id = ?`
		args = append(args, cond.UserID)
	}
	if cond.DeviceStatus == "deleted" {
		where += ` AND deleted_at <> ?`
		args = append(args, 0)
	} else if cond.DeviceStatus != "" && cond.DeviceStatus != "allDevices" {
		where += ` AND device_status = ? AND deleted_at = ?`
		args = append(args, cond.DeviceStatus, "0000-00-00 00:00:00.000")
	}
	if cond.IpLocation != "" && cond.IpLocation != "0" {
		where += ` AND ip_location = ?`
		args = append(args, cond.IpLocation)
	}
	if cond.NodeType > 0 {
		where += ` AND node_type = ?`
		args = append(args, cond.NodeType)
	}
	if cond.AreaID != "" {
		where += ` AND area_id = ?`
		args = append(args, cond.AreaID)
	}
	if cond.ExternalIp != "" {
		where += ` AND external_ip = ?`
		args = append(args, cond.ExternalIp)
	}

	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	} else {
		where += fmt.Sprintf(` ORDER BY device_status DESC, node_type DESC, cumulative_profit DESC`)
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 3000
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
		`SELECT * FROM %s %s LIMIT %d OFFSET %d`, tableNameDeviceInfo, where, limit, offset,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

func GetDeviceInfoListByKey(ctx context.Context, cond *model.DeviceInfo, option QueryOption) ([]*model.DeviceInfo, int64, error) {
	var args []interface{}
	where := `WHERE device_id <> '' AND active_status = 1`
	if cond.DeviceID != "" {
		where += ` AND device_id = ?`
		args = append(args, cond.DeviceID)
	}
	if cond.UserID != "" {
		where += ` AND user_id = ?`
		args = append(args, cond.UserID)
	} else {
		where += ` AND user_id <> ''`
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

func TranslateIPLocation(ctx context.Context, info *model.DeviceInfo, lang model.Language) {
	if lang == model.LanguageEN || lang == "" {
		return
	}

	if info.ExternalIp == "" {
		return
	}

	locationEnTable := fmt.Sprintf("%s_%s", tableNameLocation, model.LanguageEN)
	locationCnTable := fmt.Sprintf("%s_%s", tableNameLocation, lang)
	query := fmt.Sprintf(`select lc.* from %s lc join %s le on lc.ip = le.ip where le.ip = ? limit 1`, locationCnTable, locationEnTable)

	var location model.Location
	err := DB.QueryRowxContext(ctx, query, info.ExternalIp).StructScan(&location)
	if err != nil {
		log.Errorf("query %s location %s: %v", info.ExternalIp, locationCnTable, err)
		return
	}

	info.Continent = location.Continent
	info.Province = location.Province
	info.Country = location.Country
	info.City = location.City
	info.IpLocation = ContactIPLocation(location, lang)
}

func ContactIPLocation(loc model.Location, lang model.Language) string {
	var unknown string
	switch lang {
	case model.LanguageCN:
		unknown = "未知"
	default:
		unknown = "Unknown"
	}

	cf := func(in string) string {
		if in == "" {
			return unknown
		}
		return in
	}

	return fmt.Sprintf("%s-%s-%s-%s", cf(loc.Continent), cf(loc.Country), cf(loc.Province), cf(loc.City))
}

func GenerateDeviceMapInfo(infos []*model.DeviceInfo, lang model.Language, noHideIP ...bool) []map[string]interface{} {
	var out []map[string]interface{}
	mapLocationExit := make(map[float64]float64)
	for _, info := range infos {
		if info.DeviceStatus == "offline" {
			continue
		}

		Latitude, ok := mapLocationExit[info.Longitude]
		mapLocationExit[info.Longitude] = info.Latitude
		if ok && Latitude == info.Latitude {
			info.Latitude += formatter.RandFloat64() / 10000
			info.Longitude += formatter.RandFloat64() / 10000
		}

		loc, err := GetCacheLocation(context.Background(), info.ExternalIp, lang)
		if err == nil && loc.City != "" {
			info.IpCity = loc.City
		}

		//TranslateIPLocation(context.Background(), info, lang)
		if len(noHideIP) == 0 {
			maskDeviceIPAddress(info)
		}

		out = append(out, map[string]interface{}{
			"node_id":  info.DeviceID,
			"name":     info.IpCity,
			"nodeType": info.NodeType,
			"ip":       info.ExternalIp,
			"value":    []float64{info.Latitude, info.Longitude},
		})

	}
	return out

}

func maskDeviceIPAddress(deviceInfo *model.DeviceInfo) *model.DeviceInfo {
	eIp := strings.Split(deviceInfo.ExternalIp, ".")
	if len(eIp) > 3 {
		deviceInfo.ExternalIp = eIp[0] + "." + "xxx" + "." + "xxx" + "." + eIp[3]
	}
	iIp := strings.Split(deviceInfo.InternalIp, ".")
	if len(iIp) > 3 {
		deviceInfo.InternalIp = iIp[0] + "." + "xxx" + "." + "xxx" + "." + iIp[3]
	}
	return deviceInfo
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
		`UPDATE %s SET user_id = :user_id, updated_at = now(), bound_at = now(), bind_status = :bind_status WHERE device_id = :device_id`, tableNameDeviceInfo),
		deviceInfo)
	return err
}

func UpdateDeviceName(ctx context.Context, deviceInfo *model.DeviceInfo) error {
	_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET updated_at = now(),device_name = :device_name WHERE device_id = :device_id`, tableNameDeviceInfo),
		deviceInfo)
	return err
}

func BulkAddDeviceInfo(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	statement := fmt.Sprintf(
		`INSERT IGNORE INTO %s (
                	device_id, node_type, device_name, user_id, system_version,  active_status,network_info, external_ip, internal_ip, ip_location, device_rank,
                	ip_country, ip_province, ip_city, latitude, longitude, mac_location, cpu_usage, memory_usage, cpu_cores, memory, disk_usage, disk_space,
                	device_status, device_status_code, io_system, online_time, today_online_time, today_profit, yesterday_profit, seven_days_profit, month_profit,
                	cumulative_profit, bandwidth_up, bandwidth_down,download_traffic,upload_traffic, created_at, updated_at, bound_at,cache_count,retrieval_count, 
       				area_id, income_incr, last_seen, app_type
                	)
				VALUES (
					:device_id, :node_type, :device_name, :user_id,  :system_version, :active_status,:network_info, :external_ip, :internal_ip, :ip_location, :device_rank,
					:ip_country, :ip_province, :ip_city, :latitude, :longitude, :mac_location,:cpu_usage, :memory_usage, :cpu_cores, :memory, :disk_usage, :disk_space,
					:device_status, :device_status_code, :io_system, :online_time, :today_online_time, :today_profit,:yesterday_profit, :seven_days_profit, :month_profit,
					:cumulative_profit, :bandwidth_up, :bandwidth_down,:download_traffic,:upload_traffic, now(), now(),:bound_at,:cache_count,:retrieval_count, 
       				:area_id, :income_incr, :last_seen, :app_type
				)`, tableNameDeviceInfo,
	)
	_, err := DB.NamedExecContext(ctx, statement, deviceInfos)
	return err
}

func BulkUpsertDeviceInfo(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	statement := upsertDeviceInfoStatement()
	_, err := DB.NamedExecContext(ctx, statement, deviceInfos)
	return err
}

func BulkInsertOrUpdateDeviceStatus(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	statement := fmt.Sprintf(
		`INSERT INTO %s (
                	device_id, node_type, device_name, user_id, system_version,  active_status,network_info, external_ip, internal_ip, ip_location, app_type,
                	ip_country, ip_province, ip_city, latitude, longitude, mac_location, cpu_usage, memory_usage, cpu_cores, memory, disk_usage, disk_space,
                	device_status, device_status_code, io_system, online_time, today_online_time, today_profit, yesterday_profit, seven_days_profit, month_profit, area_id,
                	cumulative_profit, bandwidth_up, bandwidth_down,download_traffic,upload_traffic, created_at, updated_at, bound_at,cache_count,retrieval_count, nat_type, income_incr
                	)
				VALUES (
					:device_id, :node_type, :device_name, :user_id,  :system_version, :active_status,:network_info, :external_ip, :internal_ip, :ip_location, :app_type,
					:ip_country, :ip_province, :ip_city, :latitude, :longitude, :mac_location,:cpu_usage, :memory_usage, :cpu_cores, :memory, :disk_usage, :disk_space,
					:device_status, :device_status_code, :io_system, :online_time, :today_online_time, :today_profit,:yesterday_profit, :seven_days_profit, :month_profit, :area_id,
					:cumulative_profit, :bandwidth_up, :bandwidth_down,:download_traffic,:upload_traffic, now(), now(),:bound_at,:cache_count,:retrieval_count, :nat_type, :income_incr
				) ON DUPLICATE KEY UPDATE  device_status = VALUES(device_status), device_status_code = VALUES(device_status_code), cumulative_profit = values(cumulative_profit), updated_at = now()`, tableNameDeviceInfo,
	)
	_, err := DB.NamedExecContext(ctx, statement, deviceInfos)
	return err
}

func BulkUpdateDeviceInfo(ctx context.Context, deviceInfos []*model.DeviceInfo) error {
	insertStatement := fmt.Sprintf(
		`INSERT INTO %s (
                	device_id, node_type, device_name, user_id, system_version,  active_status,network_info, external_ip, internal_ip, ip_location,
                	ip_country, ip_province, ip_city, latitude, longitude, mac_location, cpu_usage, memory_usage, cpu_cores, memory, disk_usage, disk_space,
                	device_status, device_status_code, io_system, online_time, today_online_time, yesterday_online_time, today_profit, yesterday_profit, seven_days_profit, month_profit, area_id,
                	cumulative_profit, bandwidth_up, bandwidth_down,download_traffic,upload_traffic, created_at, updated_at, bound_at,cache_count,retrieval_count, nat_type, income_incr
                	)
				VALUES (
					:device_id, :node_type, :device_name, :user_id,  :system_version, :active_status,:network_info, :external_ip, :internal_ip, :ip_location,
					:ip_country, :ip_province, :ip_city, :latitude, :longitude, :mac_location,:cpu_usage, :memory_usage, :cpu_cores, :memory, :disk_usage, :disk_space,
					:device_status, :device_status_code, :io_system, :online_time, :today_online_time, :yesterday_online_time, :today_profit,:yesterday_profit, :seven_days_profit, :month_profit, :area_id,
					:cumulative_profit, :bandwidth_up, :bandwidth_down,:download_traffic,:upload_traffic, now(), now(),:bound_at,:cache_count,:retrieval_count, :nat_type, :income_incr
				)`, tableNameDeviceInfo,
	)
	updateStatement := ` ON DUPLICATE KEY UPDATE today_online_time = VALUES(today_online_time), today_profit = VALUES(today_profit), yesterday_profit = VALUES(yesterday_profit),seven_days_profit = VALUES(seven_days_profit),
month_profit = VALUES(month_profit), yesterday_online_time = VALUES(yesterday_online_time), updated_at = now()`
	_, err := DB.NamedExecContext(ctx, insertStatement+updateStatement, deviceInfos)
	return err
}

func upsertDeviceInfoStatement() string {
	insertStatement := fmt.Sprintf(
		`INSERT INTO %s (
                	device_id, node_type, device_name, user_id, system_version,  active_status,network_info, external_ip, internal_ip, ip_location, last_seen, app_type,
                	ip_country, ip_province, ip_city, latitude, longitude, mac_location, cpu_usage, cpu_cores, cpu_info, memory_usage, memory, disk_usage, disk_space, titan_disk_space, titan_disk_usage,
                	device_status, device_status_code, io_system, online_time, today_online_time, today_profit, yesterday_profit, seven_days_profit, month_profit, area_id,
                	cumulative_profit, bandwidth_up, bandwidth_down,download_traffic,upload_traffic, created_at, updated_at, bound_at,cache_count,retrieval_count, nat_type, income_incr, is_test_node,
                	asset_succeeded_count, asset_failed_count, retrieve_succeeded_count, retrieve_failed_count, project_count, project_succeeded_count, project_failed_count)
				VALUES (
					:device_id, :node_type, :device_name, :user_id,  :system_version, :active_status,:network_info, :external_ip, :internal_ip, :ip_location, :last_seen, :app_type,
					:ip_country, :ip_province, :ip_city, :latitude, :longitude, :mac_location,:cpu_usage, :cpu_cores, :cpu_info, :memory_usage, :memory, :disk_usage, :disk_space, :titan_disk_space, :titan_disk_usage,
					:device_status, :device_status_code, :io_system, :online_time, :today_online_time, :today_profit,:yesterday_profit, :seven_days_profit, :month_profit, :area_id,
					:cumulative_profit, :bandwidth_up, :bandwidth_down,:download_traffic,:upload_traffic, now(), now(),:bound_at,:cache_count,:retrieval_count, :nat_type, :income_incr, :is_test_node,
				    :asset_succeeded_count, :asset_failed_count, :retrieve_succeeded_count, :retrieve_failed_count, :project_count, :project_succeeded_count, :project_failed_count
				)`, tableNameDeviceInfo,
	)
	updateStatement := ` ON DUPLICATE KEY UPDATE node_type = VALUES(node_type), active_status = VALUES(active_status),
				system_version = VALUES(system_version), network_info = VALUES(network_info), cumulative_profit = VALUES(cumulative_profit),  last_seen = VALUES(last_seen), app_type = VALUES(app_type),
				external_ip = VALUES(external_ip), internal_ip = VALUES(internal_ip), ip_location = VALUES(ip_location), ip_country = VALUES(ip_country), 
				ip_province = VALUES(ip_province), ip_city = VALUES(ip_city),latitude = VALUES(latitude), longitude = VALUES(longitude), mac_location = VALUES(mac_location), area_id = VALUES(area_id),
				online_time = VALUES(online_time),cpu_usage = VALUES(cpu_usage), cpu_cores = VALUES(cpu_cores), cpu_info = VALUES(cpu_info), memory_usage = VALUES(memory_usage), memory = VALUES(memory), nat_type = VALUES(nat_type), income_incr = VALUES(income_incr),
				disk_usage = VALUES(disk_usage), disk_space = VALUES(disk_space), titan_disk_usage = VALUES(titan_disk_usage), titan_disk_space = VALUES(titan_disk_space), 
			    device_status = VALUES(device_status), device_status_code = VALUES(device_status_code) ,io_system = VALUES(io_system), bandwidth_up = VALUES(bandwidth_up),
				bandwidth_down = VALUES(bandwidth_down),download_traffic = VALUES(download_traffic),upload_traffic = VALUES(upload_traffic), updated_at = now(),bound_at = VALUES(bound_at),cache_count = VALUES(cache_count),retrieval_count = VALUES(retrieval_count),
				is_test_node = VALUES(is_test_node), asset_succeeded_count = VALUES(asset_succeeded_count), asset_failed_count = VALUES(asset_failed_count), retrieve_succeeded_count = VALUES(retrieve_succeeded_count), retrieve_failed_count = VALUES(retrieve_failed_count), 
				project_count = VALUES(project_count), project_succeeded_count = VALUES(project_succeeded_count), project_failed_count = VALUES(project_failed_count)`
	return insertStatement + updateStatement
}

func SumFullNodeInfoFromDeviceInfo(ctx context.Context) (*model.FullNodeInfo, error) {
	queryStatement := fmt.Sprintf(`
	SELECT count( device_id ) AS total_node_count ,  
			 SUM(IF(node_type = 1, 1, 0)) AS edge_count, 
			 SUM(IF(node_type = 1 AND device_status_code = 1, 1, 0)) AS online_edge_count, 
			 SUM(IF(node_type = 2 AND device_status_code = 1, 1, 0)) AS online_candidate_count, 
       		 SUM(IF(node_type = 2, 1, 0)) AS candidate_count,
			 ROUND(SUM(cache_count),0) as t_upstream_file_count,
       		 SUM(if(device_status_code = 1, 1, 0)) as online_node_count,
			 ROUND( SUM(if(device_status_code = 1, 1, 0))/count( device_id )*100,2) AS t_node_online_ratio,
       		 ROUND(SUM( disk_space),4) AS total_storage, 
			 ROUND(SUM( disk_usage*disk_space/100),4) AS storage_used, 
			 ROUND(SUM( titan_disk_space),2) AS titan_disk_space, 
			 ROUND(SUM( titan_disk_usage),2) AS titan_disk_usage, 
      		 ROUND(SUM(if(device_status_code = 1, bandwidth_up, 0)),2) AS total_upstream_bandwidth, 
			 ROUND(SUM(if(device_status_code = 1, bandwidth_down, 0)),2) AS total_downstream_bandwidth,
			 ROUND(SUM(if(device_status_code = 1, cpu_cores, 0)),0) as cpu_cores,
			 ROUND(SUM(if(device_status_code = 1, memory, 0)),0) as memory,
			 COUNT(distinct external_ip) as ip_count
		     FROM %s`, tableNameDeviceInfo)

	var out model.FullNodeInfo
	if err := DB.QueryRowxContext(ctx, queryStatement).StructScan(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UserDeviceProfile struct {
	CumulativeProfit float64 `json:"cumulative_profit" db:"cumulative_profit"`
	YesterdayProfit  float64 `json:"yesterday_profit" db:"yesterday_profit"`
	TodayProfit      float64 `json:"today_profit" db:"today_profit"`
	SevenDaysProfit  float64 `json:"seven_days_profit" db:"seven_days_profit"`
	MonthProfit      float64 `json:"month_profit" db:"month_profit"`
	NodeType         *int    `json:"node_type" db:"node_type"`
	TotalNum         int64   `json:"total_num" db:"total_num"`
	OnlineNum        int64   `json:"online_num" db:"online_num"`
	OfflineNum       int64   `json:"offline_num" db:"offline_num"`
	AbnormalNum      int64   `json:"abnormal_num" db:"abnormal_num"`
	TotalBandwidth   float64 `json:"total_bandwidth" db:"total_bandwidth"`
}

func CountUserDeviceInfo(ctx context.Context, userID string) (*UserDeviceProfile, error) {
	queryStatement := fmt.Sprintf(`SELECT COALESCE(sum(cumulative_profit),0) as cumulative_profit, COALESCE(sum(yesterday_profit),0) as yesterday_profit, 
COALESCE(sum(today_profit),0) as today_profit, count(distinct  node_type) as node_type, COALESCE(sum(seven_days_profit),0) as seven_days_profit, COALESCE(sum(month_profit),0) as month_profit, count(*) as total_num, 
count(IF(device_status = 'online', 1, NULL)) as online_num ,count(IF(device_status = 'offline', 1, NULL)) as offline_num, 
count(IF(device_status = 'abnormal', 1, NULL)) as abnormal_num, COALESCE(sum(bandwidth_up),0) as total_bandwidth from %s where user_id = ? and active_status = 1;`, tableNameDeviceInfo)

	var out UserDeviceProfile
	if err := DB.QueryRowxContext(ctx, queryStatement, userID).StructScan(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func SetDeviceUserIdToCache(ctx context.Context, deviceId, userId, areaId string) error {
	key := fmt.Sprintf("TITAN::DEVICEUSERS::%s", areaId)
	_, err := RedisCache.HSet(ctx, key, deviceId, userId).Result()
	return err
}

func GetAllDeviceUserIdFromCache(ctx context.Context, areaId string) (map[string]string, error) {
	key := fmt.Sprintf("TITAN::DEVICEUSERS::%s", areaId)
	return RedisCache.HGetAll(ctx, key).Result()
}

func SetMultipleDeviceUserIdToCache(ctx context.Context, areaId string, keyVal map[string]string) error {
	key := fmt.Sprintf("TITAN::DEVICEUSERS::%s", areaId)
	_, err := RedisCache.HSet(ctx, key, keyVal).Result()
	return err
}

func GetDeviceInfo(ctx context.Context, deviceId string) (*model.DeviceInfo, error) {
	var deviceInfo model.DeviceInfo
	query := fmt.Sprintf("SELECT * FROM %s where device_id = '%s'", tableNameDeviceInfo, deviceId)
	err := DB.QueryRowxContext(ctx, query).StructScan(&deviceInfo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoRow
	}

	if err != nil {
		log.Errorf("getDeviceInfo %v", err)
		return nil, err
	}
	return &deviceInfo, nil
}

func UpdateDeviceInfoDailyUser(ctx context.Context, deviceId, userId string) error {
	_, err := DB.ExecContext(context.Background(), "update device_info_daily set user_id = ? where device_id = ? and user_id = ''", userId, deviceId)
	return err
}

func GetDeviceInfoListByIds(ctx context.Context, deviceIds []string) ([]*model.DeviceInfo, error) {
	var out []*model.DeviceInfo
	query, args, err := sqlx.In(fmt.Sprintf(
		`SELECT * FROM %s WHERE device_id IN (?)`, tableNameDeviceInfo), deviceIds)
	if err != nil {
		return nil, err
	}

	query = DB.Rebind(query)
	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetDeviceInfoById(ctx context.Context, deviceId string) model.DeviceInfo {
	var deviceInfo model.DeviceInfo
	query := fmt.Sprintf("SELECT * FROM %s where device_id = '%s'", tableNameDeviceInfo, deviceId)
	err := DB.QueryRowxContext(ctx, query).StructScan(&deviceInfo)
	if err != nil {
		log.Errorf("getDeviceInfo %v", err)
	}
	return deviceInfo
}

func GetPlainDeviceInfoByIds(ctx context.Context, deviceIds []string) ([]*model.PlainDeviceInfo, error) {
	query := fmt.Sprintf(`
			select  device_id, device_name, device_status_code, cumulative_profit, nat_type, node_type, ip_location, CONCAT(
    	SUBSTRING_INDEX(external_ip, '.', 1), 
    	'.xxx.xxx.', 
    	SUBSTRING_INDEX(external_ip, '.', -1)
  		) AS external_ip, system_version, io_system from device_info where device_id in (?)`)

	query, args, err := sqlx.In(query, deviceIds)
	if err != nil {
		return nil, err
	}

	var out []*model.PlainDeviceInfo
	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func OnlineIPCounts(ctx context.Context) (map[string]interface{}, error) {
	query := `select external_ip, count(device_id) from device_info where device_status_code = 1 group by external_ip`

	out := make(map[string]interface{})
	rows, err := DB.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ip string
	var count int32

	for rows.Next() {
		err := rows.Scan(&ip, &count)
		if err != nil {
			return nil, err
		}

		out[ip] = count
	}

	return out, nil
}

func SetOnlineIPCountsToCache(ctx context.Context, data map[string]interface{}) error {
	key := fmt.Sprintf("TITAN::ONLINEIPCOUNTS")

	_, err := RedisCache.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	_, err = RedisCache.HSet(ctx, key, data).Result()
	return err
}

func GetOnlineIPCountsFromCache(ctx context.Context, ip string) (int64, error) {
	key := fmt.Sprintf("TITAN::ONLINEIPCOUNTS")

	result, err := RedisCache.HGet(ctx, key, ip).Result()
	if err == redis.Nil {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetNodesInfo(ctx context.Context, option QueryOption) (int64, []model.NodesInfo, error) {
	where := `WHERE device_id <> '' AND active_status = 1`
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
	err := DB.GetContext(ctx, &total, fmt.Sprintf(
		`SELECT count(distinct(user_id)) FROM %s %s`, tableNameDeviceInfo, where,
	))
	if err != nil {
		return 0, nil, err
	}
	var nodeInfo []model.NodesInfo
	query := fmt.Sprintf("SELECT node_type,user_id,COUNT(device_id) AS node_count,ROUND(sum(disk_space) ,2) as disk_space,ROUND(SUM(bandwidth_up),2) as bandwidth_up FROM %s %s GROUP BY user_id ORDER BY node_count DESC LIMIT %d OFFSET %d",
		tableNameDeviceInfo, where, limit, offset)
	err = DB.SelectContext(ctx, &nodeInfo, query)
	if err != nil {
		log.Errorf("GetNodesInfo %v", err)
		return 0, nil, err
	}
	return total, nodeInfo, nil
}

func SetDeviceProfileFromCache(ctx context.Context, deviceId string, data map[string]string) error {
	key := fmt.Sprintf("TITAN::NODE::PROFILE::%s", deviceId)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = RedisCache.Set(ctx, key, val, time.Minute*60).Result()
	return err
}

func GetDeviceProfileFromCache(ctx context.Context, deviceId string) (map[string]string, error) {
	key := fmt.Sprintf("TITAN::NODE::PROFILE::%s", deviceId)
	result, err := RedisCache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)

	err = json.Unmarshal([]byte(result), &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func SumAllUsersReward(ctx context.Context, eligibleOnlineMinutes int) ([]*model.UserReward, error) {
	query := `select user_id,
      sum(if(node_type = 2, cumulative_profit, 0)) as l1_reward,
      sum(if(node_type = 1, cumulative_profit, 0)) as l2_reward,
      sum(if(node_type = 1, today_profit,0)) as reward,
      sum(if(node_type = 1, online_incentive_profit, 0)) as online_incentive_reward,
      count(if(online_time >= ?, true, null)) as eligible_device_count,
      count(device_id) as device_count
		from device_info  where user_id <> '' GROUP BY user_id`

	var out []*model.UserReward
	err := DB.SelectContext(ctx, &out, query, eligibleOnlineMinutes)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetCountryCount 获取国家数量
func GetCountryCount(ctx context.Context) (int64, error) {
	var out int64
	query := `SELECT COUNT(DISTINCT(ip_country)) FROM device_info WHERE device_status_code = 1`

	err := DB.GetContext(ctx, &out, query)
	if err != nil {
		return 0, err
	}

	return out, nil
}

func GetTotalNodeStats(ctx context.Context, areaId string) (*model.TotalNodeStats, error) {
	query := `select 
	sum(if (node_type = 1 , cumulative_profit, 0)) as edge_rewards, 
	sum(if (node_type = 2 , cumulative_profit, 0)) as candidate_rewards,
	sum(if (device_status_code = 1, 1, 0)) as online_nodes,
	COUNT(DISTINCT external_ip) as total_ips,
	COUNT(DISTINCT CASE WHEN device_status_code = 1 THEN external_ip END) as online_ips,
	sum(if(app_type = 1 and device_status_code = 1, 1, 0)) AS online_app_nodes,
    sum(if(app_type = 2 and device_status_code = 1, 1, 0)) AS online_win_nodes,
	sum(if(app_type = 3 and device_status_code = 1, 1, 0)) AS online_mac_nodes,
	sum(titan_disk_space) as titan_disk_space,
	sum(titan_disk_usage) as titan_disk_usage
	from device_info`

	if areaId != "" {
		query += fmt.Sprintf(` where area_id = '%s'`, areaId)
	}

	var out model.TotalNodeStats
	err := DB.GetContext(ctx, &out, query)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func GetNodeIPChangedRecords(ctx context.Context, id string, option QueryOption) (int64, []*model.NodeIPHistory, error) {
	args := []interface{}{id}

	subQuery := `select device_id as node_id, external_ip, date(time) as date from device_info_daily WHERE device_id = ? and external_ip <> ''`

	countQuery := fmt.Sprintf(`SELECT count(1) FROM (%s) t`, subQuery+` group by external_ip`)

	var total int64
	err := DB.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return 0, nil, err
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 10
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	pagination := ` limit ? offset ?`
	args = append(args, limit, offset)

	query := fmt.Sprintf(`select * from (%s) t `, subQuery+` group by external_ip`) + pagination
	var out []*model.NodeIPHistory

	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return 0, nil, err
	}

	return total, out, err
}

func GetWorkerdNodes(ctx context.Context, areaId, nodeId string, option QueryOption) (int64, []*model.WorkerdNode, error) {
	var (
		where = `WHERE project_count > 0 `
		args  []interface{}
	)

	if areaId != "" {
		where += ` AND area_id = ?`
		args = append(args, areaId)
	}

	if nodeId != "" {
		where += ` AND device_id = ?`
		args = append(args, nodeId)
	}

	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	} else {
		where += fmt.Sprintf(` ORDER BY project_count DESC`)
	}

	countQuery := `SELECT count(1) FROM device_info ` + where

	var total int64
	err := DB.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return 0, nil, err
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 10
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	where += ` limit ? offset ?`
	args = append(args, limit, offset)

	var out []*model.WorkerdNode
	query := `SELECT device_id as node_id, project_count, project_succeeded_count, project_failed_count FROM device_info ` + where

	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return 0, nil, err
	}

	return total, out, err
}

func GetQualitiesNodes(ctx context.Context, areaId, nodeId string, option QueryOption) (int64, []*model.QualitiesNode, error) {
	var (
		where = `WHERE 1=1 `
		args  []interface{}
	)

	if areaId != "" {
		where += ` AND area_id = ?`
		args = append(args, areaId)
	}

	if nodeId != "" {
		where += ` AND device_id = ?`
		args = append(args, nodeId)
	}

	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	} else {
		where += fmt.Sprintf(` ORDER BY cache_count DESC`)
	}

	countQuery := `SELECT count(1) FROM device_info ` + where

	var total int64
	err := DB.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return 0, nil, err
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 10
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	where += ` limit ? offset ?`
	args = append(args, limit, offset)

	var out []*model.QualitiesNode
	query := `SELECT device_id as node_id, cache_count, asset_succeeded_count, asset_failed_count,
       bandwidth_up, retrieval_count, retrieve_succeeded_count, retrieve_failed_count, bandwidth_down
       FROM device_info ` + where

	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return 0, nil, err
	}

	return total, out, err
}

func GetIPNodeCount(ctx context.Context, ip, areaId string, option QueryOption) (int64, []*model.IPNodeCount, error) {
	var (
		where = `WHERE external_ip <> '' `
		args  []interface{}
	)

	if ip != "" {
		where += ` AND external_ip = ?`
		args = append(args, ip)
	}

	if areaId != "" {
		where += ` AND area_id = ?`
		args = append(args, areaId)
	}

	subQuery := `select DISTINCT(external_ip), area_id, sum(if(device_status_code = 1, 1,0)) as online_node_count, count(device_id) as total_node_count from device_info `

	countQuery := fmt.Sprintf(`SELECT count(1) FROM (%s) t`, subQuery+where+` group by external_ip`)

	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	} else {
		where += fmt.Sprintf(` ORDER BY online_node_count DESC`)
	}

	var total int64
	err := DB.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return 0, nil, err
	}

	limit := option.PageSize
	offset := option.Page
	if option.PageSize <= 0 {
		limit = 10
	}
	if option.Page > 0 {
		offset = limit * (option.Page - 1)
	}

	where += ` limit ? offset ?`
	args = append(args, limit, offset)

	query := fmt.Sprintf(`select * from (%s) t `, subQuery+` group by external_ip`) + where

	var out []*model.IPNodeCount

	err = DB.SelectContext(ctx, &out, query, args...)
	if err != nil {
		return 0, nil, err
	}

	return total, out, err
}
