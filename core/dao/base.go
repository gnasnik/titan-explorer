package dao

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/go-redis/redis/v9"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-module/carbon/v2"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

var (
	// DB reference to database
	DB *sqlx.DB
	// RedisCache  redis caching instance
	RedisCache *redis.Client
)

const (
	maxOpenConnections = 60
	connMaxLifetime    = 120
	maxIdleConnections = 30
	connMaxIdleTime    = 20
)

var ErrNoRow = fmt.Errorf("no matching row found")

func Init(cfg *config.Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("database url not setup")
	}

	db, err := sqlx.Connect("mysql", cfg.DatabaseURL)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(maxOpenConnections)
	db.SetConnMaxLifetime(connMaxLifetime * time.Second)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxIdleTime(connMaxIdleTime * time.Second)

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		return err
	}

	DB = db
	RedisCache = client
	return nil
}

type QueryOption struct {
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	Order      string         `json:"order"`
	OrderField string         `json:"order_field"`
	StartTime  string         `json:"start_time"`
	EndTime    string         `json:"end_time" `
	UserID     string         `json:"user_id"`
	Lang       model.Language `json:"-"`
}

func QueryMaxDeviceDailyInfo(ctx context.Context, deviceIds []string, start, end string) (map[string]*model.DeviceInfoDaily, error) {
	query := fmt.Sprintf(`
			select 
			max(user_id) as user_id, 
			max(device_id) as device_id, 
			max(nat_ratio) as nat_ratio, 
			max(disk_usage) as disk_usage, 
			max(disk_space) as disk_space, 
			max(latency) as latency, 
			max(pkg_loss_ratio) as pkg_loss_ratio, 
			max(bandwidth_up) as bandwidth_up, 
			max(bandwidth_down) as bandwidth_down, 
			max(time) as time, 
			max(hour_income) as income,
			max(online_time) as online_time,
			max(upstream_traffic) as upstream_traffic,
			max(downstream_traffic) as downstream_traffic,
			max(retrieval_count) as retrieval_count,
			max(block_count) as block_count
			from device_info_hour where device_id in (?) and time >= ? and time < ? GROUP BY device_id`)

	query, args, err := sqlx.In(query, deviceIds, start, end)
	if err != nil {
		return nil, err
	}

	rows, err := DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]*model.DeviceInfoDaily)

	for rows.Next() {
		var dailyDevice model.DeviceInfoDaily
		if err := rows.StructScan(&dailyDevice); err != nil {
			log.Errorf("struct scan: %v", err)
			continue
		}

		out[dailyDevice.DeviceID] = &dailyDevice
	}

	return out, nil
}

func GetQueryDataList(sqlClause string, args ...interface{}) ([]map[string]string, error) {
	rows, err := DB.Query(sqlClause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	dataList := make([]map[string]string, 0)
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		data := make(map[string]string)
		for i, col := range values {
			key := columns[i]
			key = strings.ToLower(key)
			data[key] = string(col)

		}
		dataList = append(dataList, data)
	}

	return dataList, nil
}

func OptionHandle(startTime, endTime string) QueryOption {
	option := QueryOption{
		StartTime: startTime,
		EndTime:   endTime,
	}
	if startTime == "" {
		option.StartTime = carbon.Now().SubDays(14).StartOfDay().String()
	}
	if endTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	} else {
		end, _ := time.Parse(formatter.TimeFormatDateOnly, endTime)
		end = end.Add(24 * time.Hour).Add(-time.Second)
		option.EndTime = end.Format(formatter.TimeFormatDatetime)
	}

	return option
}
