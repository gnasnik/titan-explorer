package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/golang-module/carbon/v2"
)

type (
	// SyncHourDataReq 同步小时时间请求
	SyncHourDataReq struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	}
)

func GetStorageHourHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	start := c.Query("from")
	end := c.Query("to")
	startTime := time.Now()

	// areaId := GetDefaultTitanCandidateEntrypointInfo()
	// schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	// if err != nil {
	// 	c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
	// 	return
	// }
	// Info, err := schedulerClient.GetUserInfo(c.Request.Context(), userId)
	// if err != nil {
	// 	log.Errorf("api GetUserInfo: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 	return
	// }
	Info, err := dao.GetUserByUsername(c.Request.Context(), userId)
	var infos []*model.UserInfo
	var userInfo model.UserInfo
	userInfo.UserId = userId
	userInfo.TotalSize = Info.TotalStorageSize
	userInfo.UsedSize = Info.UsedStorageSize
	userInfo.TotalBandwidth = Info.TotalTraffic
	userInfo.PeakBandwidth = Info.PeakBandwidth
	userInfo.DownloadCount = Info.DownloadCount
	userInfo.Time = startTime
	userInfo.CreatedAt = time.Now()
	userInfo.UpdatedAt = time.Now()
	infos = append(infos, &userInfo)
	e := dao.BulkUpsertStorageHours(c.Request.Context(), infos)
	if err != nil {
		log.Errorf("create user info hour: %v", e)
	}
	m := queryStorageHourly(c.Request.Context(), userId, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

func GetStorageDailyHandler(c *gin.Context) {
	userId := c.Query("user_id")
	start := c.Query("from")
	end := c.Query("to")
	m := QueryStorageDaily(c.Request.Context(), userId, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": m,
	}))
}

func queryStorageHourly(ctx context.Context, userId, startTime, endTime string) []*dao.UserInfoRes {
	option := dao.QueryOption{
		StartTime: startTime,
		EndTime:   endTime,
	}
	if option.StartTime == "" {
		option.StartTime = carbon.Now().StartOfHour().SubHours(25).String()
	}
	if option.EndTime == "" {
		option.EndTime = carbon.Now().String()
	} else {
		end, _ := time.Parse(formatter.TimeFormatDateOnly, endTime)
		end = end.Add(1 * time.Hour).Add(-time.Second)
		option.EndTime = end.Format(formatter.TimeFormatDatetime)
	}

	list, err := dao.GetStorageInfoHourList(ctx, userId, option)
	if err != nil {
		log.Errorf("queryStorageHourly GetStorageInfoHourList: %v", err)
		return nil
	}
	return list
}

func QueryStorageDaily(ctx context.Context, userId, startTime, endTime string) []*dao.UserInfoRes {
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
		end, _ := time.Parse(formatter.TimeFormatDateOnly, endTime)
		end = end.Add(24 * time.Hour).Add(-time.Second)
		option.EndTime = end.Format(formatter.TimeFormatDatetime)
	}
	list, err := dao.GetStorageInfoDaysList(ctx, userId, option)
	if err != nil {
		log.Errorf("QueryStorageDaily GetStorageInfoDaysList: %v", err)
		return nil
	}

	return list
}

// GetStorageHourV2Handler 获取存储每小时的信息
func GetStorageHourV2Handler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	// 获取用户的文件hash
	list, err := dao.GetUserDashboardInfos(c.Request.Context(), userId, time.Now())
	if err != nil {
		log.Errorf("GetUserDashboardInfos: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": list,
	}))
}

// SyncHourData 同步小时数据
func SyncHourData(c *gin.Context) {
	var (
		req           SyncHourDataReq
		areaIDs       []string
		wg            = new(sync.WaitGroup)
		trafficMaps   = new(sync.Map)
		bandwidthMaps = new(sync.Map)
		ahss          []model.AssetStorageHour
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg": err,
		})
		return
	}

	startTime := time.Unix(req.Start, 0)
	endTime := time.Unix(req.End, 0)

	_, maps, err := GetAndStoreAreaIDs()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg": err,
		})
		return
	}
	for _, v := range maps {
		areaIDs = append(areaIDs, v...)
	}

	for _, v := range areaIDs {
		wg.Add(1)
		go func(v string) {
			defer wg.Done()

			scli, err := GetSchedulerClient(c, v)
			if err != nil {
				log.Error(fmt.Errorf("get client of scheduler error:%w", err))
				return
			}
			infos, err := scli.GetDownloadResultsFromAssets(c, nil, startTime, endTime)
			if err != nil {
				log.Error(err)
				return
			}
			log.Debug("length of infos", len(infos))
			// 取出每个hash的最大值
			for _, v := range infos {
				log.Debug(v.Hash, v.PeakBandwidth, v.TotalTraffic)
				storeTfOrBw(trafficMaps, v.Hash, v.TotalTraffic)
				storeTfOrBw(bandwidthMaps, v.Hash, v.PeakBandwidth)
			}
		}(v)
	}
	wg.Wait()

	trafficMaps.Range(func(key, value any) bool {
		ahs := model.AssetStorageHour{TimeStamp: startTime.Add(time.Hour).Unix()}
		hash, ok := key.(string)
		if !ok {
			return true
		}
		ahs.Hash = hash
		tf, ok := value.(int64)
		if !ok {
			return true
		}
		ahs.TotalTraffic = tf
		if bv, ok := bandwidthMaps.LoadAndDelete(hash); ok {
			if bd, ok := bv.(int64); ok {
				ahs.PeakBandwidth = bd
			}
		}
		ahss = append(ahss, ahs)

		return true
	})

	c.JSON(http.StatusOK, gin.H{
		"msg": ahss,
	})
}

func storeTfOrBw(maps *sync.Map, key string, value int64) {
	if oldValue, ok := maps.Load(key); ok {
		ov, _ := oldValue.(int64)
		if ov >= value {
			return
		}
	}

	maps.Store(key, value)
}
