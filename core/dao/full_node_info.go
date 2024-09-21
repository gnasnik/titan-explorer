package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/go-redis/redis/v9"
	"time"
)

var tableNameFullNodeInfo = "full_node_info"

const (
	FullNodeInfoKeyExpiration = 0
	FullNodeInfoKey           = "TITAN::FULL_NODE_INFO"
)

func UpsertFullNodeInfo(ctx context.Context, fullNodeInfo *model.FullNodeInfo) error {
	_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (t_node_online_ratio, t_average_replica, t_upstream_file_count, candidate_count, online_candidate_count, edge_count, online_edge_count, total_storage, titan_disk_space, titan_disk_usage, storage_used, total_upstream_bandwidth, 
                total_downstream_bandwidth, total_carfile, total_carfile_size, retrieval_count, total_node_count,online_node_count,  f_backups_from_titan, cpu_cores, memory, ip_count, time, created_at) 
		VALUES (:t_node_online_ratio, :t_average_replica, :t_upstream_file_count, :candidate_count, :online_candidate_count, :edge_count, :online_edge_count, :total_storage, :storage_used, :titan_disk_space, :titan_disk_usage,
		    :total_upstream_bandwidth, :total_downstream_bandwidth, :total_carfile, :total_carfile_size, :retrieval_count, :total_node_count, :online_node_count, :f_backups_from_titan, :cpu_cores, :memory, :ip_count, :time, :created_at) 
		 ON DUPLICATE KEY UPDATE t_node_online_ratio = VALUES(t_node_online_ratio), t_average_replica = VALUES(t_average_replica), t_upstream_file_count = VALUES(t_upstream_file_count), candidate_count = VALUES(candidate_count), 
		online_candidate_count = VALUES(online_candidate_count),  online_edge_count = VALUES(online_edge_count),edge_count = VALUES(edge_count), total_storage = VALUES(total_storage), storage_used = VALUES(storage_used),  
		titan_disk_space  = VALUES(titan_disk_space), titan_disk_usage = VALUES(titan_disk_usage), total_upstream_bandwidth = VALUES(total_upstream_bandwidth), total_downstream_bandwidth = VALUES(total_downstream_bandwidth), 
		total_carfile = VALUES(total_carfile), f_backups_from_titan=VALUES(f_backups_from_titan), cpu_cores = VALUES(cpu_cores), memory = VALUES(memory), ip_count = VALUES(ip_count), total_carfile_size = VALUES(total_carfile_size), 
		retrieval_count = VALUES(retrieval_count), total_node_count = VALUES(total_node_count), online_node_count = VALUES(online_node_count)`, tableNameFullNodeInfo),
		fullNodeInfo)
	return err
}

func CacheFullNodeInfo(ctx context.Context, fullNodeInfo *model.FullNodeInfo) error {
	bytes, err := json.Marshal(fullNodeInfo)
	if err != nil {
		return err
	}
	_, err = RedisCache.Set(ctx, FullNodeInfoKey, bytes, FullNodeInfoKeyExpiration).Result()
	return err
}

func GetCacheFullNodeInfo(ctx context.Context) (*model.FullNodeInfo, error) {
	out := &model.FullNodeInfo{}
	bytes, err := RedisCache.Get(ctx, FullNodeInfoKey).Bytes()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func GetFullNodeInfoList(ctx context.Context, cond *model.FullNodeInfo, option QueryOption) ([]*model.FullNodeInfo, int64, error) {
	var args []interface{}
	where := `WHERE 1 = 1`
	if option.Order != "" && option.OrderField != "" {
		where += fmt.Sprintf(` ORDER BY %s %s`, option.OrderField, option.Order)
	} else {
		where += fmt.Sprintf(" ORDER BY time DESC")
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
	var out []*model.FullNodeInfo

	err := DB.GetContext(ctx, &total, fmt.Sprintf(
		`SELECT count(*) FROM %s %s`, tableNameFullNodeInfo, where,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	err = DB.SelectContext(ctx, &out, fmt.Sprintf(
		`SELECT * FROM %s %s LIMIT %d OFFSET %d`, tableNameFullNodeInfo, where, limit, offset,
	), args...)
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

type FullNodeDaysInfo struct {
	Date                     string  `json:"date" db:"date"`
	TotalNodeCount           int64   `db:"total_node_count" json:"total_node_count"`
	OnlineNodeCount          int64   `db:"online_node_count" json:"online_node_count"`
	SPNodeCount              int64   `db:"sp_node_count" json:"sp_node_count"`
	L1NodeCount              int64   `db:"l1_count" json:"l1_count"`
	OnlineL1NodeCount        int64   `db:"online_l1_count" json:"online_l1_count"`
	L2NodeCount              int64   `db:"l2_count" json:"l2_count"`
	OnlineL2NodeCount        int64   `db:"online_l2_count" json:"online_l2_count"`
	TUpstreamFileCount       int64   `db:"t_upstream_file_count" json:"t_upstream_file_count"`
	TotalStorage             float64 `db:"total_storage" json:"total_storage"`
	StorageUsed              float64 `db:"storage_used" json:"storage_used"`
	TotalUpstreamBandwidth   float64 `db:"total_upstream_bandwidth" json:"total_upstream_bandwidth"`
	TotalDownstreamBandwidth float64 `db:"total_downstream_bandwidth" json:"total_downstream_bandwidth"`
}

func QueryNodesDailyInfo(startTime, endTime string) []*FullNodeDaysInfo {
	list, err := GetNodesDaysList(context.Background(), OptionHandle(startTime, endTime))
	if err != nil {
		log.Errorf("QueryNodesDailyInfo GetNodesDaysList: %v", err)
		return nil
	}

	return list
}

func GetNodesDaysList(ctx context.Context, option QueryOption) ([]*FullNodeDaysInfo, error) {
	sqlClause := fmt.Sprintf(`select date_format(time, '%%Y-%%m-%%d') as date, 
	total_node_count, 
	online_node_count, 
	candidate_count as l1_count,
	online_candidate_count as online_l1_count,
	edge_count as l2_count,  
	online_edge_count as online_l2_count,  
	f_node_count as sp_node_count,
	total_upstream_bandwidth, total_downstream_bandwidth, total_storage,storage_used, t_upstream_file_count
 	from %s where time>='%s' and time<='%s' group by date order by date ASC`, tableNameFullNodeInfo, option.StartTime, option.EndTime)
	var out []*FullNodeDaysInfo
	err := DB.SelectContext(ctx, &out, sqlClause)
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		return handleNodesDaysList(option.StartTime[0:10], option.EndTime[0:10], out), err
	}
	return out, err
}

func handleNodesDaysList(start, end string, in []*FullNodeDaysInfo) []*FullNodeDaysInfo {
	startTime, _ := time.Parse(formatter.TimeFormatDateOnly, start)
	endTime, _ := time.Parse(formatter.TimeFormatDateOnly, end)
	oneDay := 24 * time.Hour
	deviceInDate := make(map[string]*FullNodeDaysInfo)
	var out []*FullNodeDaysInfo
	for _, data := range in {
		deviceInDate[data.Date] = data
	}
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(formatter.TimeFormatDateOnly)
		val, ok := deviceInDate[key]
		if !ok {
			val = &FullNodeDaysInfo{}
		}
		val.Date = startTime.Format(formatter.TimeFormatMD)
		out = append(out, val)
		startTime = startTime.Add(oneDay)
	}

	return out

}
