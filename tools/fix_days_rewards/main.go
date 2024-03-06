package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/golang-module/carbon/v2"
	"github.com/spf13/viper"
	"log"
	"time"
)

func main() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("reading config file: %v\n", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshaling config file: %v\n", err)
	}

	if err := dao.Init(&cfg); err != nil {
		log.Fatalf("initital: %v\n", err)
	}

	ctx := context.Background()
	devices, _ := getDeviceIds(ctx)

	for _, device := range devices {
		// carbon.Tomorrow().StartOfDay().Carbon2Time()
		for startTime := carbon.CreateFromDate(2024, 03, 01); startTime.Carbon2Time().Before(carbon.CreateFromDate(2024, 03, 06).Carbon2Time()); startTime = startTime.AddDay() {
			starT := startTime.StartOfDay()
			endT := startTime.EndOfDay()

			UpdateDailyIncome(ctx, device.DeviceID, starT, endT)
		}
	}

	fmt.Println("finished")
}

func getDeviceIds(ctx context.Context) ([]*model.DeviceInfo, error) {
	query := fmt.Sprintf(`select device_id, cumulative_profit from device_info where month_profit <> cumulative_profit`)

	var out []*model.DeviceInfo
	if err := dao.DB.SelectContext(ctx, &out, query); err != nil {
		log.Fatal(err)
	}

	return out, nil
}

func queryIncome(ctx context.Context, deviceId string, start, end carbon.Carbon) (float64, error) {
	st := start.StartOfDay().String()
	et := end.EndOfDay().String()

	query := fmt.Sprintf(`select ifnull(max(hour_income),0) from device_info_hour where device_id = '%s' and time >= '%s' and time < '%s' order by time desc`, deviceId, st, et)

	var income float64
	err := dao.DB.GetContext(ctx, &income, query)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	return income, err
}

func queryDaily(ctx context.Context, deviceId string, time string) (*model.DeviceInfoDaily, error) {
	qury := fmt.Sprintf(`select * from device_info_daily  where device_id = '%s' and DATE_FORMAT(time, '%%Y-%%m-%%d') = '%s'`, deviceId, time)

	var out model.DeviceInfoDaily
	err := dao.DB.GetContext(ctx, &out, qury)

	if err != nil {
		return nil, err
	}

	return &out, nil
}

func UpdateDailyIncome(ctx context.Context, deviceId string, start, end carbon.Carbon) {
	todayIncome, err := queryIncome(ctx, deviceId, start, end)
	if err != nil {
		fmt.Printf("queryIncome: %v %s %v %v\n", err, deviceId, start, end)
		return
	}

	startEpoch := carbon.CreateFromDate(2024, 03, 01)
	ends := start.SubDay()
	beforeDayIncome, err := queryIncome(ctx, deviceId, startEpoch, ends)
	if err != nil {
		fmt.Printf("queryIncome: %v %s %v %v\n", err, deviceId, start, end)
		return
	}

	//fmt.Println("time: ", start, "income: ", todayIncome, "before", beforeDayIncome)

	sub := todayIncome - beforeDayIncome
	dateTime := start.Carbon2Time().Format(time.DateOnly)

	dayIncome, err := queryDaily(ctx, deviceId, dateTime)
	if err != nil {
		// fmt.Println("queryDaily err: ", err, "device", deviceId, "time", dateTime)
		return
	}

	if dayIncome.Income != sub {
		fmt.Println("================> need update:", deviceId, dateTime, dayIncome.Income, "==>", sub)

		update := fmt.Sprintf(`update device_info_daily set income = %f where device_id = '%s' and DATE_FORMAT(time, '%%Y-%%m-%%d') = '%s' `, sub, deviceId, dateTime)
		result, err := dao.DB.ExecContext(ctx, update)
		if err != nil {
			log.Fatal(err)
		}

		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Println("update daily income success")
		}

	}

}
