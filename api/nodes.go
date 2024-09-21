package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Filecoin-Titan/titan/api/types"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/filecoin"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/oprds"
	"github.com/gnasnik/titan-explorer/core/statistics"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/go-redis/redis/v9"
	"github.com/golang-module/carbon/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func CacheAllAreas(ctx context.Context, info []string) error {
	key := fmt.Sprintf("TITAN::AREAS")

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// expiration := time.Minute * 5
	_, err = dao.RedisCache.Set(ctx, key, data, 0).Result()
	if err != nil {
		log.Errorf("set areas info: %v", err)
	}

	return nil
}

func GetAllAreasFromCache(ctx context.Context) ([]string, error) {
	key := fmt.Sprintf("TITAN::AREAS")
	result, err := dao.RedisCache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var out []string
	err = json.Unmarshal([]byte(result), &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetAllAreas(c *gin.Context) {
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"areas": []string{},
	}))
	//
	//areas, err := GetAllAreasFromCache(c.Request.Context())
	//if err == nil {
	//	c.JSON(http.StatusOK, respJSON(JsonObject{
	//		"areas": areas,
	//	}))
	//	return
	//}
	//
	//areas, err = dao.GetAllAreaFromDeviceInfo(c.Request.Context())
	//if err != nil {
	//	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	//	return
	//}
	//
	//err = CacheAllAreas(c.Request.Context(), areas)
	//if err != nil {
	//	log.Errorf("cache areas: %v", err)
	//}
	//
	//c.JSON(http.StatusOK, respJSON(JsonObject{
	//	"areas": areas,
	//}))
}

var (
	ChainHeadKey           = "TITAN::FILECOIN::CHAINHEAD"
	ChainHeadKeyExpiration = 10 * time.Second
)

func GetBlockHeightHandler(c *gin.Context) {
	lastTipSet, err := getChainHead(c.Request.Context())
	if err == nil {
		ts := filecoin.GetTimestampByHeight(lastTipSet.Height)
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"height":    lastTipSet.Height,
			"countDown": time.Now().Unix() - ts,
		}))
		return
	}

	tipSet, err := filecoin.ChainHead(config.Cfg.FilecoinRPCServerAddress)
	if err != nil {
		log.Errorf("get chain head: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if err := setChainHead(c.Request.Context(), tipSet); err != nil {
		log.Errorf("set chain head: %v", err)
	}

	ts := filecoin.GetTimestampByHeight(tipSet.Height)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"height":    tipSet.Height,
		"countDown": time.Now().Unix() - ts,
	}))
}

func getChainHead(ctx context.Context) (*filecoin.TipSet, error) {
	result, err := dao.RedisCache.Get(ctx, ChainHeadKey).Result()
	if err != nil {
		return nil, err
	}

	var ts filecoin.TipSet
	err = json.Unmarshal([]byte(result), &ts)
	if err != nil {
		return nil, err
	}

	return &ts, nil
}

func setChainHead(ctx context.Context, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	_, err = dao.RedisCache.Set(ctx, ChainHeadKey, data, ChainHeadKeyExpiration).Result()
	if err != nil {
		log.Errorf("set chain head: %v", err)
	}

	return nil
}

func GetIndexInfoHandler(c *gin.Context) {
	fullNodeInfo, err := dao.GetCacheFullNodeInfo(c.Request.Context())
	if err != nil {
		list, _, err := dao.GetFullNodeInfoList(c.Request.Context(), &model.FullNodeInfo{}, dao.QueryOption{Page: 1, PageSize: 1})
		if err != nil {
			log.Errorf("get full node info list: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		fullNodeInfo = list[0]
	}
	c.JSON(http.StatusOK, respJSON(fullNodeInfo))
}

func GetUserDeviceProfileHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	info := &model.DeviceInfo{
		UserID:       userId,
		DeviceID:     c.Query("device_id"),
		DeviceStatus: c.Query("device_status"),
	}

	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	option := dao.QueryOption{
		Page:      page,
		PageSize:  pageSize,
		StartTime: c.Query("from"),
		EndTime:   c.Query("to"),
	}

	if option.StartTime == "" {
		option.StartTime = carbon.Now().SubDays(6).StartOfDay().String()
	}
	if option.EndTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	} else {
		option.EndTime = carbon.Parse(option.EndTime).EndOfDay().String()
	}

	userDeviceProfile, err := dao.CountUserDeviceInfo(c.Request.Context(), info.UserID)
	if err != nil {
		log.Errorf("database CountUserDeviceInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	m, err := dao.GetUserIncome(info, option)
	if err != nil {
		log.Errorf("database GetUserIncome: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	data := toDeviceStatistic(option.StartTime, option.EndTime, m)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"profile":     userDeviceProfile,
		"series_data": data,
	}))
}

func GetUserDevicesCountHandler(c *gin.Context) {
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
		option.StartTime = time.Now().AddDate(0, 0, -6).Format(formatter.TimeFormatDateOnly)
	}
	if option.EndTime == "" {
		option.EndTime = time.Now().Format(formatter.TimeFormatDateOnly)
	}

	userDeviceProfile, err := dao.CountUserDeviceInfo(c.Request.Context(), info.UserID)
	if err != nil {
		log.Errorf("GetUserDevicesCountHandler CountUserDeviceInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"profile": userDeviceProfile,
	}))
}

func toDeviceStatistic(start, end string, data map[string]map[string]interface{}) []*dao.DeviceStatistics {
	startTime := carbon.Parse(start)
	endTime := carbon.Parse(end)

	var out []*dao.DeviceStatistics
	for st := startTime; st.Lte(endTime); st = st.AddDay() {
		key := st.StdTime().Format(formatter.TimeFormatDateOnly)
		_, ok := data[key]
		if !ok {
			out = append(out, &dao.DeviceStatistics{
				Date: key,
			})
			continue
		}
		out = append(out, &dao.DeviceStatistics{
			Date:   key,
			Income: data[key]["income"].(float64),
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
	} else {
		option.EndTime = carbon.Parse(endTime).EndOfDay().String()
	}

	condition := &model.DeviceInfoDaily{
		DeviceID: deviceID,
	}

	list, err := dao.GetDeviceInfoDailyListAppendDays(context.Background(), condition, option)
	if err != nil {
		log.Errorf("database GetDeviceInfoDailyList: %v", err)
		return nil
	}

	return list
}

func queryDeviceDailyByUserId(userId string, option dao.QueryOption) []*dao.DeviceStatistics {
	if option.StartTime == "" {
		option.StartTime = carbon.Now().SubDays(14).StartOfDay().String()
	}
	if option.EndTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	} else {
		option.EndTime = carbon.Parse(option.EndTime).EndOfDay().String()
	}
	condition := &model.DeviceInfoDaily{
		UserID: userId,
	}

	list, err := dao.GetNodesInfoDailyList(context.Background(), condition, option)
	if err != nil {
		log.Errorf("database GetNodesInfoDailyList: %v", err)
		return nil
	}

	return list
}

func queryDeviceStatisticHourly(deviceID, startTime, endTime string) []*dao.DeviceStatistics {
	option := dao.QueryOption{
		StartTime: startTime,
		EndTime:   endTime,
	}
	if option.StartTime == "" {
		option.StartTime = carbon.Now().StartOfHour().SubHours(24).String()
	}
	if option.EndTime == "" {
		option.EndTime = carbon.Now().EndOfDay().String()
	} else {
		option.EndTime = carbon.Parse(endTime).EndOfDay().String()
	}

	condition := &model.DeviceInfoHour{
		DeviceID: deviceID,
	}

	list, err := dao.GetDeviceInfoHourList(context.Background(), condition, option)
	if err != nil {
		log.Errorf("database GetDeviceInfoHourList: %v", err)
		return nil
	}

	return list
}

type DeviceRewardRule struct {
	NatRule       []Rule `json:"nat_rule"`
	NodeCountRule []Rule `json:"node_count_rule"`
	BandwidthRule []Rule `json:"bandwidth_rule"`
}

type Rule struct {
	Name     string  `json:"name"`
	FullName string  `json:"full_name"`
	Score    float64 `json:"score"`
	Current  bool    `json:"current"`
}

func GetQueryInfoHandler(c *gin.Context) {
	key := c.Query("key")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	lang := model.Language(c.GetHeader("Lang"))

	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	deviceInfos, total, err := dao.GetDeviceInfoListByKey(c.Request.Context(), &model.DeviceInfo{UserID: key}, option)
	if err != nil {
		log.Errorf("get device by user id info list: %v", err)
	}

	for _, device := range deviceInfos {
		allTime := decimal.NewFromInt(time.Now().Unix() - device.BoundAt.Unix()).Div(decimal.NewFromFloat(60))
		offLineTime := allTime.Sub(decimal.NewFromFloat(device.OnlineTime))
		onLineRate := decimal.NewFromFloat(device.OnlineTime).Div(allTime)
		device.OffLineTime, _ = offLineTime.Round(0).Float64()
		device.OnLineRate, _ = onLineRate.Round(2).Float64()
		if device.OnlineTime/24 > 24 {
			device.UnLockProfit = device.CumulativeProfit
		} else {
			device.LockProfit = device.CumulativeProfit
		}
		nodeInfo, err := getNodeInfoByScheduler(c.Request.Context(), device.DeviceID, device.AreaID)
		if err != nil {
			continue
		}
		device.Mx = nodeInfo.Mx
		device.PenaltyProfit = nodeInfo.PenaltyProfit
	}

	if total > 0 {
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"list":  maskIPAddress(deviceInfos),
			"total": total,
			"type":  "user_id",
		}))
		return
	}

	deviceInfo := dao.GetDeviceInfoById(context.Background(), key)
	if deviceInfo.DeviceID != "" {
		deviceInfo.CumulativeProfit = deviceInfo.CumulativeProfit + deviceInfo.OnlineIncentiveProfit
		deviceInfos = append(deviceInfos, &deviceInfo)
	} else {
		device, err := getDeviceInfoFromSchedulerAndInsert(c.Request.Context(), key, "")
		if err != nil {
			c.JSON(http.StatusOK, respJSON(JsonObject{
				"type": "wrong key",
			}))
			return
		}
		deviceInfo = *device
		deviceInfos = append(deviceInfos, device)
	}

	for _, device := range deviceInfos {
		dao.TranslateIPLocation(c.Request.Context(), device, lang)
		allTime := decimal.NewFromInt(time.Now().Unix() - device.BoundAt.Unix()).Div(decimal.NewFromFloat(60))
		offLineTime := allTime.Sub(decimal.NewFromFloat(device.OnlineTime))
		onLineRate := decimal.NewFromFloat(device.OnlineTime).Div(allTime)
		device.OffLineTime, _ = offLineTime.Round(0).Float64()
		device.OnLineRate, _ = onLineRate.Round(2).Float64()
		if device.OnlineTime/24 > 24 {
			device.UnLockProfit = device.CumulativeProfit
		} else {
			device.LockProfit = device.CumulativeProfit
		}
		nodeInfo, err := getNodeInfoByScheduler(c.Request.Context(), device.DeviceID, device.AreaID)
		if err != nil {
			continue
		}
		device.Mx = nodeInfo.Mx
		device.PenaltyProfit = nodeInfo.PenaltyProfit

		maskLocation(device, lang)
	}

	ipDeviceCounts, err := dao.GetOnlineIPCountsFromCache(c.Request.Context(), deviceInfo.ExternalIp)
	if err != nil {
		log.Errorf("get count ip device: %v", err)
	}

	rule := DeviceRewardRule{
		NatRule:       getNatRule(deviceInfo.NATType, lang),
		NodeCountRule: getNodeCountRule(ipDeviceCounts),
		BandwidthRule: getBandwidth(deviceInfo.BandwidthUp),
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  maskIPAddress(deviceInfos),
		"rule":  rule,
		"total": total,
		"type":  "node_id",
	}))

}

func getBandwidth(bandwidthUp float64) []Rule {
	defaultRule := []Rule{
		{Name: "<5Mbps", Score: 0.8, Current: bandwidthUp < 5_000_000},
		{Name: "5Mbps<=X<30Mbps", Score: 1, Current: bandwidthUp > 5_000_000 && bandwidthUp < 30_000_000},
		{Name: "30Mbps<=X", Score: 1.2, Current: bandwidthUp > 30_000_000},
	}

	return defaultRule
}

func getNodeCountRule(count int64) []Rule {
	return []Rule{
		{Name: "5", Score: 0.1, Current: count >= 5},
		{Name: "4", Score: 0.2, Current: count == 4},
		{Name: "3", Score: 0.3, Current: count == 3},
		{Name: "2", Score: 0.4, Current: count == 2},
		{Name: "1", Score: 1.1, Current: count == 1},
	}
}

func getNatRule(natType string, lang model.Language) []Rule {
	var (
		publicIPName      = "PublicIP"
		natPublicFullName = "No NAT"
		nat1FullName      = "Full Cone"
		nat2FullName      = "Restricted Cone"
		nat3FullName      = "Port Restricted Cone"
		nat4FullName      = "Symmetric"
	)

	if lang == model.LanguageCN {
		publicIPName = "公网IP"
		natPublicFullName = fmt.Sprintf("公网IP(%s)", natPublicFullName)
		nat1FullName = fmt.Sprintf("全锥型(%s)", nat1FullName)
		nat2FullName = fmt.Sprintf("受限锥型(%s)", nat2FullName)
		nat3FullName = fmt.Sprintf("端口受限锥形(%s)", nat3FullName)
		nat4FullName = fmt.Sprintf("对称型(%s)", nat4FullName)
	}

	return []Rule{
		{Name: "NAT4", FullName: nat4FullName, Score: 0.8, Current: "SymmetricNAT" == natType || "UnknowNAT" == natType || "" == natType},
		{Name: "NAT3", FullName: nat3FullName, Score: 1.1, Current: "PortRestrictedNAT" == natType},
		{Name: "NAT2", FullName: nat2FullName, Score: 1.3, Current: "RestrictedNAT" == natType},
		{Name: "NAT1", FullName: nat1FullName, Score: 1.5, Current: "FullConeNAT" == natType},
		{Name: publicIPName, FullName: natPublicFullName, Score: 2, Current: "NoNAT" == natType},
	}

}

func maskIPAddress(in []*model.DeviceInfo) []*model.DeviceInfo {
	for _, deviceInfo := range in {
		eIp := strings.Split(deviceInfo.ExternalIp, ".")
		if len(eIp) > 3 {
			deviceInfo.ExternalIp = eIp[0] + "." + "xxx" + "." + "xxx" + "." + eIp[3]
		}
		iIp := strings.Split(deviceInfo.InternalIp, ".")
		if len(iIp) > 3 {
			deviceInfo.InternalIp = iIp[0] + "." + "xxx" + "." + "xxx" + "." + iIp[3]
		}
	}
	return in
}

func GetDeviceInfoHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username, auth := claims[identityKey].(string)

	info := &model.DeviceInfo{}
	info.UserID = c.Query("user_id")
	info.DeviceID = c.Query("device_id")
	info.IpLocation = c.Query("ip_location")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	nodeTypeStr := c.Query("node_type")
	lang := model.Language(c.GetHeader("Lang"))
	notBound := c.Query("not_bound")
	info.AreaID = c.Query("area_id")
	info.ExternalIp = c.Query("ip")

	if auth {
		user, err := dao.GetUserByUsername(c.Request.Context(), username)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		if model.UserRole(user.Role) != model.UserRoleAdmin {
			info.UserID = username
		}
	}

	if nodeTypeStr != "" {
		nodeType, _ := strconv.ParseInt(nodeTypeStr, 10, 64)
		info.NodeType = nodeType
	}
	activeStatusStr := c.Query("active_status")
	if activeStatusStr == "" {
		info.ActiveStatus = 10
	} else {
		activeStatus, _ := strconv.ParseInt(activeStatusStr, 10, 64)
		info.ActiveStatus = activeStatus
	}
	deviceStatus := c.Query("device_status")

	if deviceStatus == "online" || deviceStatus == "offline" || deviceStatus == "abnormal" || deviceStatus == "deleted" {
		info.DeviceStatus = deviceStatus
	}
	if deviceStatus == "unbinding" || deviceStatus == "unbound" {
		info.BindStatus = deviceStatus
	}
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
		NotBound:   notBound,
	}

	deviceInfos, total, err := dao.GetDeviceInfoList(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("database GetDeviceInfoList: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	offset := (option.Page - 1) * option.PageSize
	for i, deviceInfo := range deviceInfos {
		deviceInfo.DeviceRank = int64(i + 1 + offset)
		dao.TranslateIPLocation(c.Request.Context(), deviceInfo, lang)
		maskLocation(deviceInfo, lang)
	}

	if !auth {
		maskIPAddress(deviceInfos)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  deviceInfos,
		"total": total,
	}))
}

func maskLocation(d *model.DeviceInfo, lang model.Language) {
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

	if d.IpCountry == "China" || d.IpCountry == "中国" {
		d.Country = cf("")
		d.Province = cf("")
		d.City = cf("")
		d.IpLocation = fmt.Sprintf("%s-%s-%s-%s", cf(d.Continent), cf(""), cf(""), cf(""))
	}

}

func GetDeviceActiveInfoHandler(c *gin.Context) {
	//info := &model.DeviceInfo{}
	//info.UserID = c.Query("user_id")
	//pageSize, _ := strconv.Atoi(c.Query("page_size"))
	//page, _ := strconv.Atoi(c.Query("page"))
	//order := c.Query("order")
	//orderField := c.Query("order_field")
	//activeStatusStr := c.Query("active_status")
	//if activeStatusStr == "" {
	//	info.ActiveStatus = 10
	//} else {
	//	activeStatus, _ := strconv.ParseInt(activeStatusStr, 10, 64)
	//	info.ActiveStatus = activeStatus
	//}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  nil,
		"total": 0,
	}))
}

func GetDeviceStatusHandler(c *gin.Context) {
	info := &model.DeviceInfo{}
	info.UserID = c.Query("user_id")
	info.DeviceID = c.Query("device_id")
	info.DeviceStatus = c.Query("device_status")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	info.ActiveStatus = 1
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	deviceInfos, total, err := dao.GetDeviceInfoList(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("GetDeviceStatusHandler GetDeviceInfoList: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  maskIPAddress(deviceInfos),
		"total": total,
	}))
}

func GetNodesInfoHandler(c *gin.Context) {
	info := &model.DeviceInfo{}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	nodeType, _ := strconv.ParseInt(c.Query("node_type"), 10, 64)
	info.NodeType = nodeType
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}
	var total int64
	total, list, err := dao.GetNodesInfo(c.Request.Context(), option)
	if err != nil {
		log.Errorf("GetNodesInfoHandler GetNodesInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  handleNodesRank(&list, option),
		"total": total,
	}))
}

func handleNodesRank(nodes *[]model.NodesInfo, opt dao.QueryOption) *[]model.NodesInfo {
	var nodesRank []model.NodesInfo
	offset := (opt.Page - 1) * opt.PageSize
	for i, info := range *nodes {
		rank := strconv.Itoa(i + 1 + offset)
		info.Rank = rank
		nodesRank = append(nodesRank, info)
	}
	return &nodesRank
}

func GetMapInfoHandler(c *gin.Context) {
	lang := model.Language(c.GetHeader("Lang"))
	deviceId := c.Query("device_id")

	if deviceId != "" {
		mapInfo, err := dao.GetDeviceMapInfo(c.Request.Context(), lang, deviceId)
		if err != nil {
			log.Errorf("GetDeviceMapInfo: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"list": mapInfo,
		}))
		return
	}

	mapInfo, err := dao.GetMapInfoFromCache(c.Request.Context(), lang)
	if err == nil {
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"list": mapInfo,
		}))
		return
	}

	mapInfo, err = dao.GetDeviceMapInfo(c.Request.Context(), lang, deviceId)
	if err != nil {
		log.Errorf("GetDeviceMapInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	err = dao.CacheMapInfo(c.Request.Context(), mapInfo, lang)
	if err != nil {
		log.Errorf("cache mapinfo: %v", err)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list": mapInfo,
	}))
}

//maskIPAddress(deviceInfos)

func GetDeviceDiagnosisDailyByDeviceIdHandler(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	deviceID := c.Query("device_id")

	if deviceID == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	m := queryDeviceStatisticsDaily(deviceID, from, to)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

func GetDeviceDiagnosisDailyByUserIdHandler(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	userId := c.Query("user_id")
	notBound := c.Query("not_bound")

	if userId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	option := dao.QueryOption{
		StartTime: from,
		EndTime:   to,
		NotBound:  notBound,
	}

	m := queryDeviceDailyByUserId(userId, option)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

func GetDeviceDiagnosisHourHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	start := c.Query("from")
	end := c.Query("to")

	if deviceID == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	data := make([]*dao.DeviceStatistics, 0)
	data = queryDeviceStatisticHourly(deviceID, start, end)

	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceID)
	if err != nil {
		log.Errorf("get device info: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data":  data,
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

	if cond.DeviceID == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		OrderField: "created_at",
		Order:      "DESC",
	}

	list, total, err := dao.GetDeviceInfoDailyByPage(context.Background(), cond, option)
	if err != nil {
		log.Errorf("get device info daily: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

func GetDiskDaysHandler(c *gin.Context) {
	//date := c.Query("date")
	start := c.Query("from")
	end := c.Query("to")
	m := dao.QueryNodesDailyInfo(start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

func getDeviceInfoFromSchedulerAndInsert(ctx context.Context, nodeId string, areaId string) (*model.DeviceInfo, error) {
	device, err := getNodeInfoFromScheduler(ctx, nodeId, areaId)
	if err != nil {
		log.Errorf("getNodeInfoFromScheduler %v", err)
		return nil, err
	}

	fn, err := dao.GetCacheFullNodeInfo(ctx)
	if err == nil && fn != nil {
		device.DeviceRank = int64(fn.TotalNodeCount) + 1
	}

	err = dao.BulkAddDeviceInfo(ctx, []*model.DeviceInfo{device})
	if err != nil {
		log.Errorf("BulkAddDeviceInfo %v", err)
		return nil, err
	}

	return device, nil
}

func getNodeInfoFromScheduler(ctx context.Context, id string, areaId string) (*model.DeviceInfo, error) {
	// 判断节点是否存在
	next, _ := oprds.GetClient().CheckUnSyncNodeID(ctx, id)
	if !next {
		return nil, fmt.Errorf("device not found")
	}

	if areaId != "" {
		schedulerClient, err := getSchedulerClient(ctx, areaId)
		if err != nil {
			return nil, err
		}

		nodeInfo, err := schedulerClient.GetNodeInfo(ctx, id)
		if err != nil {
			oprds.GetClient().IncrUnSyncNodeID(ctx, id)
			log.Errorf("get node info error:%w", err)
			return nil, err
		}

		return statistics.ToDeviceInfo(*nodeInfo, areaId), nil
	}

	for _, schedulerClient := range statistics.Schedulers {
		nodeInfo, err := schedulerClient.Api.GetNodeInfo(ctx, id)
		if err == nil {
			return statistics.ToDeviceInfo(*nodeInfo, areaId), nil
		}
		log.Errorf("get node info error:%w", err)
	}

	oprds.GetClient().IncrUnSyncNodeID(ctx, id)

	return nil, fmt.Errorf("device not found")
}

func getNodeInfoByScheduler(ctx context.Context, id string, areaId string) (*types.NodeInfo, error) {
	// 判断节点是否存在
	next, _ := oprds.GetClient().CheckUnSyncNodeID(ctx, id)
	if !next {
		return nil, fmt.Errorf("device not found")
	}

	if areaId == "" {
		areaId = GetDefaultTitanCandidateEntrypointInfo()
	}

	schedulerClient, err := getSchedulerClient(ctx, areaId)
	if err != nil {
		return nil, err
	}

	nodeInfo, err := schedulerClient.GetNodeInfo(ctx, id)
	if err != nil {
		oprds.GetClient().IncrUnSyncNodeID(ctx, id)
		log.Errorf("get node info error:%w", err)
		return nil, err
	}

	return nodeInfo, nil
}

func GetDeviceProfileHandler(c *gin.Context) {
	type getEarningReq struct {
		NodeID string   `json:"node_id"`
		AreaID string   `json:"area_id"`
		Keys   []string `json:"keys"`
		Since  int64    `json:"since"`
	}

	var param getEarningReq
	if err := c.BindJSON(&param); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	out := make(map[string]interface{})
	out["since"] = time.Now().Unix()

	lastUpdate, err := dao.GetCacheFullNodeInfo(c.Request.Context())
	if err != nil {
		log.Errorf("get last update info: %v", err)
	}

	dataChanged := true
	if lastUpdate != nil && param.Since > 0 {
		sinceT := time.Unix(param.Since, 0)
		if lastUpdate.Time.Before(sinceT) {
			dataChanged = false
		}
	}

	deviceInfo, err := dao.GetDeviceInfo(c.Request.Context(), param.NodeID)
	if err == dao.ErrNoRow {
		device, err := getDeviceInfoFromSchedulerAndInsert(c.Request.Context(), param.NodeID, param.AreaID)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
			return
		}

		deviceInfo = device
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	response := make(map[string]interface{})

	for _, key := range param.Keys {
		switch key {
		case "epoch":
			out[key] = struct {
				Token string `json:"token"`
			}{
				Token: config.Cfg.Epoch.Token,
			}
		case "info":
			out[key] = struct {
				Status     int64   `json:"status"`
				IncomeIncr float64 `json:"income_incr"`
			}{
				IncomeIncr: deviceInfo.IncomeIncr,
				Status:     deviceInfo.DeviceStatusCode,
			}
		case "account":
			out[key] = queryAccountInfo(c.Request.Context(), deviceInfo.DeviceID, deviceInfo.UserID)
		case "income":
			if !dataChanged {
				continue
			}
			response[key] = map[string]interface{}{
				"today": deviceInfo.TodayProfit,
				"total": deviceInfo.CumulativeProfit,
			}
		case "month_incomes":
			if !dataChanged {
				continue
			}
			response[key] = queryDailyIncome(c.Request.Context(), param.NodeID)
		}
	}

	fr, err := filterResponse(c.Request.Context(), param.NodeID, response)
	if err != nil {
		log.Errorf("filter response: %v", err)
	}

	if param.Since > 0 {
		response = fr
	}

	for key, val := range response {
		out[key] = val
	}

	c.JSON(http.StatusOK, respJSON(out))
}

func filterResponse(ctx context.Context, nodeId string, response map[string]interface{}) (map[string]interface{}, error) {
	deviceCache, err := dao.GetDeviceProfileFromCache(ctx, nodeId)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	out := make(map[string]interface{})
	devHash := make(map[string]string)

	for key, val := range response {
		encodeData, err := json.Marshal(val)
		if err != nil {
			log.Errorf("encode %s: %v", key, err)
			continue
		}

		hasher := md5.New()
		hasher.Write(encodeData)
		hash := hex.EncodeToString(hasher.Sum(nil))
		checksum := deviceCache[key]

		if checksum != hash {
			out[key] = val
		}

		devHash[key] = hash
	}

	err = dao.SetDeviceProfileFromCache(ctx, nodeId, devHash)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func queryAccountInfo(ctx context.Context, deviceId, userId string) interface{} {
	account := struct {
		UserId        string `json:"user_id"`
		WalletAddress string `json:"wallet_address"`
		Code          string `json:"code"`
	}{}

	if userId == "" {
		return account
	}

	account.UserId = userId
	user, err := dao.GetUserByUsername(ctx, userId)
	if err != nil {
		log.Errorf("get user %v", err)
	}

	if user != nil {
		account.WalletAddress = user.WalletAddress
	}

	signature, err := dao.GetSignatureByNodeId(ctx, deviceId)
	if err != nil {
		log.Errorf("get signature: %v", err)
	}

	if signature != nil {
		account.Code = signature.Hash
	}

	return account
}

func queryDailyIncome(ctx context.Context, nodeId string) interface{} {
	start := carbon.Now().SubDays(30).String()

	option := dao.QueryOption{
		StartTime: start,
		EndTime:   carbon.Now().String(),
	}

	condition := &model.DeviceInfoDaily{
		DeviceID: nodeId,
	}

	list, err := dao.GetDeviceInfoDailyList(context.Background(), condition, option)
	if err != nil {
		log.Errorf("database GetDeviceInfoDailyList: %v", err)
		return nil
	}

	out := make([]interface{}, 0)
	for _, item := range list {
		out = append(out, map[string]interface{}{
			"k": item.Date,
			"v": item.Income,
		})
	}

	return out
}

func GenerateCodeHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	//message := fmt.Sprintf(`Signature for titan \n %s \n%s`, username, time.Now().Format(time.RFC3339Nano))

	hash := strings.ToUpper(uuid.NewString())

	if err := dao.AddSignature(c.Request.Context(), &model.Signature{
		Username: username,
		//Message:  message,
		Hash: hash,
	}); err != nil {
		log.Errorf("add signature: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		//"message": message,
		"code": hash,
	}))
}

func QueryDeviceCodeHandler(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidCode, c))
		return
	}

	signature, err := dao.GetSignatureByHash(c.Request.Context(), code)
	if err == dao.ErrNoRow {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidCode, c))
		return
	}

	if err != nil {
		log.Errorf("get signature: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	user, err := dao.GetUserByUsername(c.Request.Context(), signature.Username)
	if err != nil {
		log.Errorf("get user: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"user_id":        user.Username,
		"wallet_address": user.WalletAddress,
	}))
}

func CacheDeviceDistribution(ctx context.Context, info []*model.DeviceDistribution, lang model.Language) error {
	key := fmt.Sprintf("TITAN::DISTRIBUTION::%s", lang)

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	expiration := time.Minute * 5
	_, err = dao.RedisCache.Set(ctx, key, data, expiration).Result()
	if err != nil {
		log.Errorf("set areas info: %v", err)
	}

	return nil
}

func GetDeviceDistributionFromCache(ctx context.Context, lang model.Language) ([]*model.DeviceDistribution, error) {
	key := fmt.Sprintf("TITAN::DISTRIBUTION::%s", lang)
	result, err := dao.RedisCache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var out []*model.DeviceDistribution
	err = json.Unmarshal([]byte(result), &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetDeviceDistributionHandler(c *gin.Context) {
	lang := model.Language(c.GetHeader("Lang"))

	distribution, err := GetDeviceDistributionFromCache(c.Request.Context(), lang)
	if err == nil {
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"distribution": distribution,
		}))
		return
	}

	distribution, err = dao.GetDeviceDistribution(c.Request.Context(), lang)
	if err != nil {
		log.Errorf("get device distribution: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	for i, distr := range distribution {
		if distr.Country == "China" {
			distribution[i].Country = "Unknown"
		}

		if distr.Country == "中国" {
			distribution[i].Country = "未知"
		}
	}

	err = CacheDeviceDistribution(c.Request.Context(), distribution, lang)
	if err != nil {
		log.Errorf("cache distribution: %v", err)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"distribution": distribution,
	}))
}

func GetPlainDeviceInfoHandler(c *gin.Context) {
	type query struct {
		Ids []string `json:"ids"`
	}

	var req query
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if len(req.Ids) > 100 {
		c.JSON(http.StatusOK, respErrorCode(errors.LimitExceeded, c))
		return
	}

	deviceInfos, err := dao.GetPlainDeviceInfoByIds(c.Request.Context(), req.Ids)
	if err != nil {
		log.Errorf("GetPlainDeviceInfoByIds: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"devices": deviceInfos,
	}))
}

func GetDeviceOnlineIncentivesHandler(c *gin.Context) {
	deviceId := c.Query("device_id")
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("page_size"), 10, 64)

	option := dao.QueryOption{
		Page:     int(page),
		PageSize: int(size),
	}

	list, total, err := dao.GetDeviceOnlineIncentiveList(c.Request.Context(), deviceId, option)
	if err != nil {
		log.Errorf("GetDeviceOnlineIncentiveList: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

func GetIPRecordsHandler(c *gin.Context) {
	ip := c.Query("ip")
	areaId := c.Query("area_id")
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("page_size"), 10, 64)

	option := dao.QueryOption{
		Page:     int(page),
		PageSize: int(size),
	}

	total, list, err := dao.GetIPNodeCount(c.Request.Context(), ip, areaId, option)
	if err != nil {
		log.Errorf("GetIPNodeCount: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}
