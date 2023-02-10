package api

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	"github.com/golang-module/carbon/v2"
	"net/http"
	"strconv"
	"time"
)

// GetAllAreas is a function to get a slice of record(s) from device info table in database
//
//	@Summary		获取所有的地区列表
//	@Tags			矿工页
//	@Description	GetAllAreas is a handler to get a slice of record(s)from device info table in database
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		string
//	@Failure		400	{object}	errors.GenericError
//	@Router			/all_areas [get]
func GetAllAreas(c *gin.Context) {
	areas, err := dao.GetAllAreaFromDeviceInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"areas": areas,
	}))
}

// GetIndexInfoHandler is a function to get a single record from the full node info table in the redis
//
//	@Summary		获取数据总览数据
//	@Tags			首页
//
//	@Description	GetIndexInfoHandler is a function to get a single record from the full node info table in the redis
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.FullNodeInfo
//	@Failure		400	{object}	errors.GenericError
//	@Router			/get_index_info [get]
func GetIndexInfoHandler(c *gin.Context) {
	fullNodeInfo, err := dao.GetCacheFullNodeInfo(c.Request.Context())
	if err != nil {
		log.Errorf("get full node info: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}
	c.JSON(http.StatusOK, respJSON(fullNodeInfo))
}

// GetUserDeviceProfileHandler is a function to get user devices overview
//
//	@Summary		用户总览
//	@Tags			控制台
//	@Description	DeviceBindingHandler is a function to get user devices overview
//	@Accept			json
//	@Produce		json
//	@Param			user_id	query		string	true	"用户ID"
//	@Success		200		{object}	JsonObject{profile=dao.UserDeviceProfile,series_data=[]dao.DeviceStatistics}
//	@Failure		400		{object}	errors.GenericError
//	@Router			/get_user_device_profile [get]
func GetUserDeviceProfileHandler(c *gin.Context) {
	info := &model.DeviceInfo{}
	info.UserID = c.Query("user_id")
	info.DeviceID = c.Query("device_id")
	info.DeviceStatus = c.Query("device_status")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	option := dao.QueryOption{
		Page:      page,
		PageSize:  pageSize,
		StartTime: c.Query("from"),
		EndTime:   c.Query("to"),
	}

	if option.StartTime == "" {
		option.StartTime = time.Now().AddDate(0, 0, -6).Format(utils.TimeFormatYMD)
	}
	if option.EndTime == "" {
		option.EndTime = time.Now().Format(utils.TimeFormatYMD)
	}

	userDeviceProfile, err := dao.CountUserDeviceInfo(c.Request.Context(), info.UserID)
	if err != nil {
		log.Errorf("get user device profile: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrNotFound))
		return
	}

	m, err := dao.GetUserIncome(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get user income: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrNotFound))
		return
	}

	data := toDeviceStatistic(option.StartTime, option.EndTime, m)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"profile":     userDeviceProfile,
		"series_data": data,
	}))
}

func toDeviceStatistic(start, end string, data map[string]map[string]interface{}) []*dao.DeviceStatistics {
	startTime, _ := time.Parse(utils.TimeFormatYMD, start)
	endTime, _ := time.Parse(utils.TimeFormatYMD, end)

	var oneDay = 24 * time.Hour
	var out []*dao.DeviceStatistics
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		key := startTime.Format(utils.TimeFormatYMD)
		startTime = startTime.Add(oneDay)
		val, ok := data[key]
		if !ok {
			out = append(out, &dao.DeviceStatistics{
				Date: key,
			})
			continue
		}
		out = append(out, &dao.DeviceStatistics{
			Date:   key,
			Income: val["income"].(float64),
		})
	}

	return out
}

func queryDeviceStatisticsDaily(deviceID, startTime, endTime string) []*dao.DeviceStatistics {
	option := dao.QueryOption{
		StartTime: startTime,
		EndTime:   endTime,
	}
	if startTime == "" {
		option.StartTime = carbon.Now().SubDays(14).StartOfDay().String()
	}
	if endTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	}

	condition := &model.DeviceInfoDaily{
		DeviceID: deviceID,
	}

	list, err := dao.GetDeviceInfoDailyList(context.Background(), condition, option)
	if err != nil {
		log.Errorf("get incoming daily: %v", err)
		return nil
	}

	return list
}

func queryDeviceStatisticHourly(deviceID, start, end string) []*dao.DeviceStatistics {
	option := dao.QueryOption{
		StartTime: start,
		EndTime:   end,
	}
	if option.StartTime == "" {
		option.StartTime = carbon.Now().StartOfHour().SubHours(25).String()
	}
	if option.EndTime == "" {
		option.EndTime = carbon.Now().String()
	}

	condition := &model.DeviceInfoHour{
		DeviceID: deviceID,
	}

	list, err := dao.GetDeviceInfoDailyHourList(context.Background(), condition, option)
	if err != nil {
		log.Errorf("get incoming hour daily: %v", err)
		return nil
	}

	return list
}

// GetDeviceInfoHandler is a function to get a slice of record(s) from device info table in database
//
//	@Summary		获取矿工页列表
//	@Tags			矿工页
//	@Description	GetDeviceInfoHandler is a handler to get a slice of record(s) from device info table in database
//	@Accept			json
//	@Produce		json
//	@Param			user_id		query		string	false	"用户ID"
//	@Param			device_id	query		string	false	"设备ID"
//	@Param			ip_location	query		int		false	"国家地区"
//	@Param			node_type	query		int		false	"节点类型，默认全部"
//	@Param			page		query		int		false	"分页页码，默认 1"
//	@Param			page_size	query		int		false	"分页页数 默认 50"
//	@Param			order		query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field	query		string	false	"排序字段"
//	@Success		200			{object}	JsonObject{list=[]model.DeviceInfo}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_device_info [get]
func GetDeviceInfoHandler(c *gin.Context) {
	info := &model.DeviceInfo{}
	info.UserID = c.Query("user_id")
	info.DeviceID = c.Query("device_id")
	info.DeviceStatus = c.Query("device_status")
	info.IpLocation = c.Query("ip_location")
	pageSize, _ := strconv.Atoi("page_size")
	page, _ := strconv.Atoi("page")
	order := c.Query("order")
	orderField := c.Query("order_field")
	nodeType, _ := strconv.ParseInt(c.Query("node_type"), 10, 64)
	info.NodeType = int32(nodeType)
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}
	list, total, err := dao.GetDeviceInfoList(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get device info list: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

type MapReturnForSwag struct {
	Name string
	Value []float64
}

// GetMapInfoHandler is a function to get a slice of record(s) from devices map
//
//	@Summary		获取地图信息接口
//	@Tags			首页
//	@Description	GetMapInfoHandler is a function to get a slice of record(s) from devices map
//	@Accept			json
//	@Produce		json
//	@Param			device_id		query		string	true	"设备ID"
//	@Param			device_status	query		string	false	"状态"
//	@Param			page			query		string	false	"页码"
//	@Param			page_size		query		string	false	"页数"
//	@Param			order			query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field		query		string	false	"排序字段"
//	@Success		200				{object}	JsonObject{list=[]MapReturnForSwag, total=number}
//	@Failure		400				{object}	errors.GenericError
//	@Router			/get_map_info [get]
func GetMapInfoHandler(c *gin.Context) {
	info := &model.DeviceInfo{}
	info.UserID = c.Query("user_id")
	info.DeviceID = c.Query("device_id")
	info.DeviceStatus = c.Query("device_status")
	pageSize, _ := strconv.Atoi("page_size")
	page, _ := strconv.Atoi("page")
	order := c.Query("order")
	orderField := c.Query("order_field")
	nodeType, _ := strconv.ParseInt(c.Query("node_type"), 10, 64)
	info.NodeType = int32(nodeType)
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}
	list, total, err := dao.GetDeviceInfoList(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get device info list: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  dao.HandleMapInfo(list),
		"total": total,
	}))
}

// GetDeviceDiagnosisDailyHandler is a function to get a slice of record(s) from device info daily diagnosis
//
//	@Summary		获取历史数据诊断
//	@Tags			矿工页
//	@Description	GetDeviceDiagnosisDailyHandler is a function to get a slice of record(s) from device info daily diagnosis
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			from		query		string	false	"开始时间"
//	@Param			to			query		string	false	"结束时间"
//	@Success		200			{object}	JsonObject{series_data=dao.DeviceStatistics}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_diagnosis_days [get]
func GetDeviceDiagnosisDailyHandler(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	deviceID := c.Query("device_id")
	m := queryDeviceStatisticsDaily(deviceID, from, to)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

// GetDeviceDiagnosisHourHandler is a function to get a slice of record(s) from device info hourly diagnosis
//
//	@Summary		获取实时监控
//	@Tags			矿工页
//	@Description	GetDeviceDiagnosisHourHandler is a function to get a slice of record(s) from device info hourly diagnosis
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			from		query		string	false	"开始时间"
//	@Param			to			query		string	false	"结束时间"
//	@Success		200			{object}	JsonObject{series_data=dao.DeviceStatistics}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_diagnosis_hours [get]
func GetDeviceDiagnosisHourHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	//date := c.Query("date")
	start := c.Query("from")
	end := c.Query("to")
	m := queryDeviceStatisticHourly(deviceID, start, end)
	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data":  m,
		"cpu_cores":    deviceInfo.CpuCores,
		"cpu_usage":    fmt.Sprintf("%.2f", deviceInfo.CpuUsage),
		"memory":       fmt.Sprintf("%.2f", deviceInfo.Memory/float64(10<<20)),
		"memory_usage": fmt.Sprintf("%.2f", deviceInfo.MemoryUsage*deviceInfo.Memory/float64(10<<20)),
		"disk_usage":   fmt.Sprintf("%.2f", (deviceInfo.DiskUsage*deviceInfo.DiskSpace/100)/float64(10<<30)),
		"disk_space":   fmt.Sprintf("%.2f", deviceInfo.DiskSpace/float64(10<<30)),
		"disk_type":    deviceInfo.DiskType,
		"file_system":  deviceInfo.IoSystem,
		"w":            []float64{},
	}))
}

func GetDeviceInfoDailyHandler(c *gin.Context) {
	cond := &model.DeviceInfoDaily{}
	cond.DeviceID = c.Query("device_id")
	pageSize, _ := strconv.Atoi("page_size")
	page, _ := strconv.Atoi("page")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		OrderField: "created_at",
		Order:      "DESC",
	}

	list, total, err := dao.GetDeviceInfoDailyByPage(context.Background(), cond, option)
	if err != nil {
		log.Errorf("get device info daily: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}
