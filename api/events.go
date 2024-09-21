package api

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gnasnik/titan-explorer/config"

	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/gnasnik/titan-explorer/pkg/rsa"

	"github.com/Filecoin-Titan/titan/api"
	"github.com/Filecoin-Titan/titan/api/terrors"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/Filecoin-Titan/titan/node/cidutil"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/oprds"
	"github.com/gnasnik/titan-explorer/core/storage"
)

// GetDefaultTitanCandidateEntrypointInfo  specify candidate to upload file in testnet, only for storage api
func GetDefaultTitanCandidateEntrypointInfo() string {
	cfg := config.Cfg.SpecifyCandidate
	return cfg.AreaId
}

func GetCacheListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if nodeId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), nodeId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	// todo: get scheduler from area id
	resp, err := schedulerClient.GetReplicaEventsForNode(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api GetReplicaEventsForNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.List,
		"total": resp.Total,
	}))
	return
}

func GetValidationListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), nodeId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	resp, err := schedulerClient.GetValidationResults(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api GetValidationResults: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	//var validationEvents []*model.ValidationEvent
	//for _, blockInfo := range resp.ValidationResultInfos {
	//	validationEvents = append(validationEvents, toValidationEvent(blockInfo))
	//}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.ValidationResultInfos,
		"total": resp.Total,
	}))
}

func GetReplicaListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	queryStatus := c.Query("status")

	if nodeId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	var status []types.ReplicaStatus
	for _, s := range strings.Split(queryStatus, ",") {
		statusVal, _ := strconv.ParseInt(s, 10, 64)
		status = append(status, types.ReplicaStatus(statusVal))
	}

	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), nodeId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	resp, err := schedulerClient.GetReplicasForNode(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize, status)
	if err != nil {
		log.Errorf("api GetReplicaEventsForNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.NodeReplicaInfos,
		"total": resp.Total,
	}))
}

func GetProfitDetailsHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	queryStatus := c.Query("ts")

	if nodeId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	var ts []int
	for _, s := range strings.Split(queryStatus, ",") {
		statusVal, _ := strconv.ParseInt(s, 10, 64)
		ts = append(ts, int(statusVal))
	}

	deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), nodeId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	resp, err := schedulerClient.GetProfitDetailsForNode(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize, ts)
	if err != nil {
		log.Errorf("api GetReplicaEventsForNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.Infos,
		"total": resp.Total,
	}))
}

//	     _______________  ____  ___   ____________   ___    ____  ____
//		/ ___/_  __/ __ \/ __ \/   | / ____/ ____/  /   |  / __ \/  _/
//		\__ \ / / / / / / /_/ / /| |/ / __/ __/    / /| | / /_/ // /
//     ___/ // / / /_/ / _, _/ ___ / /_/ / /___   / ___ |/ ____// /
//	  /____//_/  \____/_/ |_/_/  |_\____/_____/  /_/  |_/_/   /___/

func GetAllocateStorageHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	// if userId == "" {
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 	return
	// }
	// var userInfo model.User
	// userInfo.Username = userId
	// _, err := dao.GetUserByUsername(c.Request.Context(), userInfo.Username)
	// if err == nil {
	// 	log.Info("GetUserByUsername user exists")
	// } else {
	// 	_ = dao.CreateUser(c.Request.Context(), &userInfo)
	// }

	// // areaId := GetDefaultTitanCandidateEntrypointInfo()
	// // schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	// // if err != nil {
	// // 	c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
	// // 	return
	// // }
	// // _, err = schedulerClient.AllocateStorage(c.Request.Context(), userId)
	// if err != nil {
	// 	if webErr, ok := err.(*api.ErrWeb); ok {
	// 		c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
	// 		return
	// 	}
	// 	log.Errorf("api GetValidationResults: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 	return
	// }

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// GetStorageSizeHandler 获取用户存储空间信息
// ShareAssetsHandler 获取用户存储空间信息
// @Summary 获取用户存储空间信息
// @Description 获取用户存储空间信息
// @Security ApiKeyAuth
// @Tags storage
// @Success 200 {object} JsonObject "{PeakBandwidth:0,TotalTraffic:0,TotalSize:0,UsedSize:0}"
// @Router /api/v1/storage/get_storage_size [get]
func GetStorageSizeHandler(c *gin.Context) {
	var fInfo = new(dao.UserStorageFlowInfo)

	claims := jwt.ExtractClaims(c)
	username, ok := claims[identityKey].(string)
	if !ok {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	value, err := oprds.GetClient().GetUserStorageFlowInfo(c.Request.Context(), username)
	if err != nil {
		fInfo, err = dao.GetUserStorageFlowInfo(c.Request.Context(), username)
		if err != nil {
			fInfo = new(dao.UserStorageFlowInfo)
			log.Error(err)
		} else {
			ib, _ := json.Marshal(fInfo)
			oprds.GetClient().StoreUserStorageFlowInfo(c.Request.Context(), username, string(ib))
		}
	} else {
		json.Unmarshal([]byte(value), fInfo)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"PeakBandwidth": fInfo.PeakBandwidth,
		"TotalTraffic":  fInfo.TotalTraffic,
		"TotalSize":     user.TotalStorageSize,
		"UsedSize":      user.UsedStorageSize,
	}))
}

// GetUserVipInfoHandler 判断用户是否是vip
// @Summary 判断用户是否是vip
// @Description 判断用户是否是vip
// @Security ApiKeyAuth
// @Tags storage
// @Success 200 {object} JsonObject "{vip:false}"
// @Router /api/v1/storage/get_vip_info [get]
func GetUserVipInfoHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Errorf("api GetUserInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"vip": user.EnableVIP,
		"uid": username,
	}))
	return
}

func GetUserAccessTokenHandler(c *gin.Context) {
	// UserId := c.Query("user_id")
	// claims := jwt.ExtractClaims(c)
	// UserId := claims[identityKey].(string)
	// areaId := getAreaID(c)
	// schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	// if err != nil {
	// 	c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
	// 	return
	// }
	// token, err := schedulerClient.GetUserAccessToken(c.Request.Context(), UserId)
	// if err != nil {
	// 	log.Errorf("api GetUserAccessToken: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
	// 	return
	// }
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"AccessToken": "token",
	}))
}

const FileUploadPassKey = "TITAN::FILE::PASS::%s"

func GetUploadInfoHandler(c *gin.Context) {
	var aid string
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	// ts := c.Query("ts")
	// signature := c.Query("signature")

	areaId := getAreaIDs(c)

	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId[0])
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	// var nonce string
	// if ts != "" && signature != "" {
	// 	nonce, err = dao.RedisCache.Get(c.Request.Context(), userId+ts).Result()
	// 	if err != nil {
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 		return
	// 	}
	// }

	var randomPassNonce string
	if c.Query("encrypted") == "true" {
		passKey := fmt.Sprintf(FileUploadPassKey, userId)
		randomPassNonce = string(md5Str(userId + time.Now().String()))
		if _, err = dao.RedisCache.SetEx(c.Request.Context(), passKey, randomPassNonce, 24*time.Hour).Result(); err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}

	var urlModel bool
	if c.Query("urlModel") == "true" {
		urlModel = true
	}

	res, err := schedulerClient.GetNodeUploadInfo(c.Request.Context(), userId, randomPassNonce, urlModel)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	as := strings.Split(areaId[0], "-")
	if len(as) < 2 {
		aid = areaId[0]
	} else {
		aid = as[1]
	}

	c.JSON(http.StatusOK, respJSON(gin.H{
		"AlreadyExists": res.AlreadyExists,
		"List":          res.List,
		"AreaID":        aid,
		"Log":           areaId[0],
	}))
}

func md5Str(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

// CreateAssetHandler 上传文件
// @Summary 上传文件
// @Description 上传文件
// @Security ApiKeyAuth
// @Tags storage
// @Param area_id query string false "节点区域"
// @Param asset_name query string true "文件名"
// @Param asset_cid query string true "文件cid"
// @Param node_id query string true "节点id"
// @Param asset_type query string true "文件类型"
// @Param asset_size query int64 true "文件大小"
// @Param group_id query int true "group id"
// @Success 200 {object} JsonObject "{[]{CandidateAddr:"",Token:""}}"
// @Router /api/v1/storage/create_asset [get]
func CreateAssetHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	areaIds := getAreaIDs(c)

	if len(areaIds) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// 判断用户是否存在
	user, err := dao.GetUserByUsername(c.Request.Context(), userId)
	switch err {
	case sql.ErrNoRows:
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	case nil:
	default:
		log.Errorf("CreateAssetHandler dao.GetUserByUsername() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var randomPassNonce string
	if c.Query("encrypted") == "true" {
		passKey := fmt.Sprintf(FileUploadPassKey, userId)
		randomPassNonce = dao.RedisCache.Get(c.Request.Context(), passKey).Val()
		if randomPassNonce == "" {
			log.Error("CreateAssetHandler randomPassNonce not found")
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		dao.RedisCache.Del(c.Request.Context(), passKey)
	}

	log.Debugf("CreateAssetHandler clientIP:%s, areaId:%v\n", c.ClientIP(), areaIds)

	var createAssetReq createAssetRequest
	createAssetReq.AssetName = c.Query("asset_name")
	createAssetReq.AssetCID = c.Query("asset_cid")
	createAssetReq.NodeID = c.Query("node_id")
	createAssetReq.AssetType = c.Query("asset_type")
	createAssetReq.AssetSize = formatter.Str2Int64(c.Query("asset_size"))
	createAssetReq.GroupID, _ = strconv.ParseInt(c.Query("group_id"), 10, 64)

	// 获取文件hash
	hash, err := storage.CIDToHash(createAssetReq.AssetCID)
	if err != nil {
		log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	notExistsAids, err := dao.GetUserAssetNotAreaIDs(c.Request.Context(), hash, userId, areaIds)
	if err != nil {
		log.Errorf("GetUserAssetByAreaIDs error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	}
	if len(notExistsAids) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.FileExists, c))
		return
	}
	// 判断用户存储空间是否够用
	if user.TotalStorageSize-user.UsedStorageSize < createAssetReq.AssetSize {
		c.JSON(http.StatusOK, respErrorCode(int(terrors.UserStorageSizeNotEnough), c))
		return
	}

	// 调用调度器
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaIds[0])
	if err != nil {
		log.Errorf("CreateAssetHandler getSchedulerClient error: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, gin.H{
				"code": -1,
				"err":  webErr.Code,
				"msg":  webErr.Message,
				"Log":  areaIds[0],
			})
			return
		}
	}
	createAssetRsp, err := schedulerClient.CreateAsset(c.Request.Context(), &types.CreateAssetReq{
		UserID: userId, AssetCID: createAssetReq.AssetCID, AssetSize: createAssetReq.AssetSize, NodeID: createAssetReq.NodeID, Owner: userId})
	if err != nil {
		log.Errorf("CreateAssetHandler CreateAsset error: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, gin.H{
				"code": -1,
				"err":  webErr.Code,
				"msg":  webErr.Message,
				"Log":  areaIds[0],
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			// "err":  webErr.Code,
			// "msg":  webErr.Message,
			"Log": areaIds[0],
		})
		return
	}

	if !createAssetRsp.AlreadyExists {
		if len(createAssetRsp.List) == 0 {
			log.Errorf("createAssetRsp.List: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}
	// 判断是否需要同步调度器信息
	if len(notExistsAids) > 0 {
		err = oprds.GetClient().PushSchedulerInfo(c.Request.Context(), &oprds.Payload{UserID: userId, CID: createAssetReq.AssetCID, Hash: hash, AreaID: areaIds[0]})
		if err != nil {
			log.Errorf("PushSchedulerInfo error: %v", err)
		}
	}

	// aids, _ := syncShedulers(c.Request.Context(), schedulerClient, createAssetReq.NodeID, createAssetReq.AssetCID, createAssetReq.AssetSize, areaIds)
	// aids = append(aids, areaIds[0])

	if err := dao.AddAssetAndUpdateSize(c.Request.Context(), &model.UserAsset{
		UserID:      userId,
		Hash:        hash,
		Cid:         createAssetReq.AssetCID,
		AssetName:   createAssetReq.AssetName,
		AssetType:   createAssetReq.AssetType,
		CreatedTime: time.Now(),
		TotalSize:   createAssetReq.AssetSize,
		Password:    randomPassNonce,
		GroupID:     int64(createAssetReq.GroupID),
	}, notExistsAids, areaIds[0]); err != nil {
		log.Errorf("CreateAssetHandler AddAsset error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	rsp := make([]JsonObject, len(createAssetRsp.List))
	if !createAssetRsp.AlreadyExists {
		for i, v := range createAssetRsp.List {
			rsp[i] = JsonObject{"CandidateAddr": v.UploadURL, "Token": v.Token}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": rsp,
		"Log":  areaIds[0],
	})
}

type createAssetRequest struct {
	AssetName string   `json:"asset_name" binding:"required"`
	AssetCID  string   `json:"asset_cid"`
	AreaID    []string `json:"area_id"`
	NodeID    string   `json:"node_id"`
	AssetType string   `json:"asset_type" binding:"required"`
	AssetSize int64    `json:"asset_size" binding:"required"`
	GroupID   int64    `json:"group_id"`
	Encrypted bool     `json:"encrypted"`
}

// CreateAssetPostHandler 创建文件
// @Summary 上传文件
// @Description 上传文件
// @Security ApiKeyAuth
// @Tags storage
// @Param req body createAssetRequest true "请求参数"
// @Success 200 {object} JsonObject "{[]{CandidateAddr:"",Token:""}}"
// @Router /api/v1/storage/create_asset [post]
func CreateAssetPostHandler(c *gin.Context) {
	var (
		claims          = jwt.ExtractClaims(c)
		username        = claims[identityKey].(string)
		randomPassNonce string
		createAssetReq  createAssetRequest
	)

	if err := c.ShouldBindJSON(&createAssetReq); err != nil {
		log.Errorf("CreateAssetHandler c.BindJSON() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	areaIds := getAreaIDsByArea(c, createAssetReq.AreaID)
	if len(areaIds) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	// 判断用户是否存在
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	switch err {
	case sql.ErrNoRows:
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	case nil:
	default:
		log.Errorf("CreateAssetHandler dao.GetUserByUsername() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	// 判断上传文件是否需要加密
	if c.Query("encrypted") == "true" {
		passKey := fmt.Sprintf(FileUploadPassKey, username)
		randomPassNonce = dao.RedisCache.Get(c.Request.Context(), passKey).Val()
		if randomPassNonce == "" {
			log.Error("CreateAssetHandler randomPassNonce not found")
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		dao.RedisCache.Del(c.Request.Context(), passKey)
	}

	// 获取文件hash
	log.Debugf("areaid:%v", areaIds[0])
	hash, err := storage.CIDToHash(createAssetReq.AssetCID)
	if err != nil {
		log.Errorf("CreateAssetHandler storage.CIDToHash() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// 获取文件不存在的区域列表，默认都上传成功
	notExistsAids, err := dao.GetUserAssetNotAreaIDs(c.Request.Context(), hash, username, areaIds)
	if err != nil {
		log.Errorf("GetUserAssetByAreaIDs error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	}
	if len(notExistsAids) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.FileExists, c))
		return
	}
	// 判断用户存储空间是否够用
	if user.TotalStorageSize-user.UsedStorageSize < createAssetReq.AssetSize {
		c.JSON(http.StatusOK, respErrorCode(int(terrors.UserStorageSizeNotEnough), c))
		return
	}

	// 判断所选区域(areaIds[0])是否同步成功，默认已经同步成功
	var createAssetRsp = new(types.UploadInfo)
	createAssetRsp.AlreadyExists = true
	if !dao.CheckAssetIsSyncByAreaID(c.Request.Context(), hash, areaIds[0]) {
		// 调用调度器
		schedulerClient, err := getSchedulerClient(c.Request.Context(), areaIds[0])
		if err != nil {
			log.Errorf("CreateAssetHandler getSchedulerClient error: %v", err)
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, gin.H{
					"code": -1,
					"err":  webErr.Code,
					"msg":  webErr.Message,
					"Log":  areaIds[0],
				})
				return
			}
		}
		createAssetRsp, err = schedulerClient.CreateAsset(c.Request.Context(), &types.CreateAssetReq{
			UserID: username, AssetCID: createAssetReq.AssetCID, AssetSize: createAssetReq.AssetSize, NodeID: createAssetReq.NodeID, Owner: username})
		if err != nil {
			log.Errorf("CreateAssetHandler CreateAsset error: %v", err)
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, gin.H{
					"code": -1,
					"err":  webErr.Code,
					"msg":  webErr.Message,
					"Log":  areaIds[0],
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": -1,
				"Log":  areaIds[0],
			})
			return
		}
		if !createAssetRsp.AlreadyExists {
			if len(createAssetRsp.List) == 0 {
				log.Errorf("createAssetRsp.List: %v", err)
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
				return
			}
		}
	}

	// 判断是否需要同步调度器信息
	if len(notExistsAids) > 0 {
		err = oprds.GetClient().PushSchedulerInfo(c.Request.Context(), &oprds.Payload{UserID: username, CID: createAssetReq.AssetCID, Hash: hash, AreaID: areaIds[0]})
		if err != nil {
			log.Errorf("PushSchedulerInfo error: %v", err)
		}
	}

	if err := dao.AddAssetAndUpdateSize(c.Request.Context(), &model.UserAsset{
		UserID:      username,
		Hash:        hash,
		Cid:         createAssetReq.AssetCID,
		AssetName:   createAssetReq.AssetName,
		AssetType:   createAssetReq.AssetType,
		CreatedTime: time.Now(),
		TotalSize:   createAssetReq.AssetSize,
		Password:    randomPassNonce,
		GroupID:     int64(createAssetReq.GroupID),
	}, notExistsAids, areaIds[0]); err != nil {
		log.Errorf("CreateAssetHandler AddAsset error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	rsp := make([]JsonObject, len(createAssetRsp.List))
	if !createAssetRsp.AlreadyExists {
		for i, v := range createAssetRsp.List {
			rsp[i] = JsonObject{"CandidateAddr": v.UploadURL, "Token": v.Token}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": rsp,
		"Log":  areaIds[0],
	})
}

type CidArr []string

func CreateAssetFromIPFSHandler(c *gin.Context) {
	var arr CidArr
	if err := c.BindJSON(&arr); err != nil {
		log.Errorf("CreateAssetFromIPFSHandler c.BindJSON() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

}

func ExportAssetToIPFSHandler(c *gin.Context) {

}

// func FilePassNonceHandler(c *gin.Context) {

// 	pass := rand.Reader
// 	claims := jwt.ExtractClaims(c)
// 	userId := claims[identityKey].(string)

// 	if pass == "" {
// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
// 		return
// 	}

// 	nonce := rsa.EncryptPassWithSalt(pass)

// 	_, err := dao.RedisCache.SetEx(c.Request.Context(), userId+ts, nonce, 60*time.Minute).Result()
// 	if err != nil {
// 		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
// 	}

// 	c.JSON(http.StatusOK, respJSON(JsonObject{
// 		"nonce": nonce,
// 	}))
// }

// ----- upload process --------
// 1. pass + timestamp -> nonce
// 2. metamask + nonce -> signature
// 3. signature + timestamp -> verify
// 4. get_upload_node_info + signature + timestamp -> url + token
// 5. source + token -> L1 -> cid
// 6. cid + node_id + signature -> create_asset -> nonce + asset -> db

// ----- share process --------
// 1. expire_time + asset -> token link
//

// ----- upload process --------
//  1. encrypted ? -> get_upload_info
//     yes -> randomPassNonce -> token -> savePass2redis
//     no -> upload_file_without_encryption
//  2. upload with token
//
// 3.
func FilePassVerifyHandler(c *gin.Context) {
	ts := c.Query("ts")
	signature := c.Query("signature")

	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	if ts == "" || signature == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	nonce, err := dao.RedisCache.Get(c.Request.Context(), userId+ts).Result()
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	addr, err := rsa.VerifyAddrSign(nonce, signature)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidSignature, c))
		return
	}

	if addr != userId {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// CreateKeyHandler 创建key
// @Summary 创建key
// @Description 创建key
// @Security ApiKeyAuth
// @Tags storage
// @Param key_name query string true "key name"
// @Success 200 {object} JsonObject "{key:"",secret:""}"
// @Router /api/v1/storage/create_key [get]
func CreateKeyHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	keyName := c.Query("key_name")
	// 获取apikey
	info, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}
	buf, keyStr, secretStr, err := storage.CreateAPIKeySecret(c.Request.Context(), userId, keyName, info.ApiKeys)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	err = dao.UpdateUserAPIKeys(c.Request.Context(), info.ID, buf)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"key":    keyStr,
		"secret": secretStr,
	}))
}

// DeleteKeyHandler 删除key
// @Summary 删除key
// @Description 删除key
// @Security ApiKeyAuth
// @Tags storage
// @Param key_name query string true "key name"
// @Success 200 {object} JsonObject "{msg:""}"
// @Router /api/v1/storage/delete_key [get]
func DeleteKeyHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	keyName := c.Query("key_name")

	info, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}
	if len(info.ApiKeys) > 0 {
		keyMaps, err := storage.DecodeAPIKeySecrets(info.ApiKeys)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		if _, ok := keyMaps[keyName]; !ok {
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		delete(keyMaps, keyName)
		buf, err := storage.EncodeAPIKeySecrets(keyMaps)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		err = dao.UpdateUserAPIKeys(c.Request.Context(), info.ID, buf)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "delete success",
	}))
}

// DeleteAssetHandler 删除文件
// @Summary 删除文件
// @Description 删除文件
// @Security ApiKeyAuth
// @Tags storage
// @Param area_id query string false "节点区域"
// @Param asset_cid query string true "文件cid"
// @Success 200 {object} JsonObject "{msg:""}"
// @Router /api/v1/storage/delete_asset [get]
func DeleteAssetHandler(c *gin.Context) {
	var (
		wg          = new(sync.WaitGroup)
		mu          = new(sync.Mutex)
		execAreaIds []string
	)
	claims := jwt.ExtractClaims(c)
	userID := claims[identityKey].(string)
	cid := c.Query("asset_cid")
	areaIds := getAreaIDsNoDefault(c)
	// 获取文件hash
	hash, err := storage.CIDToHash(cid)
	if err != nil {
		log.Errorf("CreateAssetHandler storage.CIDToHash() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// 获取文件信息
	areaIds, isNeedDel, err := dao.CheckUserAseetNeedDel(c.Request.Context(), hash, userID, areaIds)
	if err != nil {
		log.Errorf("get areaIds error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if len(areaIds) == 0 {
		c.JSON(http.StatusOK, respErrorCode(int(terrors.NotFound), c))
		return
	}
	// 调用scheduler接口删除文件
	for _, v := range areaIds {
		wg.Add(1)
		go func(v string) {
			defer func() {
				wg.Done()
				mu.Lock()
				execAreaIds = append(execAreaIds, v)
				mu.Unlock()
			}()
			// 判断文件是否为唯一存在的
			isOnly, err := dao.CheckUserAssetIsOnly(c.Request.Context(), hash, v)
			if err != nil {
				return
			}
			if !isOnly {
				return
			}
			scli, err := getSchedulerClient(c.Request.Context(), v)
			if err != nil {
				return
			}
			err = scli.RemoveAssetRecord(c.Request.Context(), cid)
			if err != nil {
				log.Errorf("get areaIds error: %+v", err)
			}
		}(v)
	}
	wg.Wait()
	if len(execAreaIds) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// 判断是否需要进行删除
	msg := "delete success"
	if len(areaIds) != len(execAreaIds) {
		if isNeedDel {
			isNeedDel = false
		}
		msg = "Partially deleted successfully"
	}
	err = dao.DelAssetAndUpdateSize(c.Request.Context(), hash, userID, execAreaIds, isNeedDel)
	if err != nil {
		log.Errorf("api DeleteAsset: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": msg,
	}))
}

// ShareAssetsHandler 分享文件
// @Summary 分享文件
// @Description 分享文件
// @Tags storage
// @Param user_id query string true "用户id"
// @Param area_id query string false "节点区域"
// @Param asset_cid query string true "文件cid"
// @Success 200 {object} JsonObject "{asset_cid: "",redirect:"",url:{}}"
// @Router /api/v1/storage/share_asset [get]
func ShareAssetsHandler(c *gin.Context) {
	var (
		areaId  string
		areaIds []string
	)
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	cid := c.Query("asset_cid")
	if c.Query("area_id") != "" {
		areaIds = getAreaIDs(c)
	}

	// 判断是否超过流量限额
	ok, err := checkUserTotalFlow(c.Request.Context(), userId)
	if err != nil {
		log.Error("check user total flow error: ", cid)
	}
	if !ok {
		c.JSON(http.StatusOK, respErrorCode(errors.OutTotalFlow, c))
		return
	}

	hash, err := cidutil.CIDToHash(cid)
	if err != nil {
		log.Error("Invalid asset CID: ", cid)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	// 如果用户指定了区域，则先判断区域是否存在
	if len(areaIds) > 0 {
		exist, err := dao.CheckUserAssetIsInAreaID(c.Request.Context(), userId, hash, areaIds[0])
		if err != nil {
			if err == sql.ErrNoRows {
				log.Errorf("sql error no rows")
				c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			} else {
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			}
			return
		}
		if !exist {
			log.Errorf("no exist")
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		areaId, err = dao.GetOneAreaIDByAreaID(c.Request.Context(), userId, hash, areaId)
		if err != nil {
			log.Errorf("get one areaid error:%v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		}
	} else {
		// 获取用户文件所有的区域
		areaIDs, err := dao.GetUserAssetAreaIDs(c.Request.Context(), hash, userId)
		if err != nil {
			log.Errorf("get user assest areaids error:%w", err)
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		if len(areaIDs) == 0 {
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		// 获取用户的访问的ip
		ip, err := GetIPFromRequest(c.Request)
		if err != nil {
			log.Errorf("get user's ip of request error:%w", err)
			// c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			// return
			areaId = areaIDs[0]
		} else {
			areaId, err = GetNearestAreaID(c.Request.Context(), ip, areaIDs)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
				return
			}
		}

	}
	// 获取文件信息
	userAsset, err := dao.GetUserAssetDetail(c.Request.Context(), hash, userId)
	if err != nil {
		log.Error("Failed to get user asset: ", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	// dao.AddVisitCount(c.Request.Context(), hash)

	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	var ret []string
	if userAsset.Password != "" {
		urls, err := schedulerClient.ShareEncryptedAsset(c.Request.Context(), userId, cid, userAsset.Password, time.Time{})
		if err != nil {
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
				return
			}
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		ret = urls
	} else {
		urls, err := schedulerClient.ShareAssets(c.Request.Context(), userId, []string{cid}, time.Time{})
		if err != nil {
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
				return
			}
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		ret = urls[cid]
	}

	for i := range ret {
		ret[i] = fmt.Sprintf("%s&filename=%s", ret[i], userAsset.AssetName)
	}

	// 成功的时候，下载量+1
	oprds.GetClient().IncrAssetHourDownload(c.Request.Context(), hash)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"asset_cid": cid,
		"size":      userAsset.TotalSize,
		"url":       ret,
		"redirect":  false,
	}))
}

func OpenAssetHandler(c *gin.Context) {
	var (
		cid     = c.Query("asset_cid")
		userId  = c.Query("user_id")
		areaIds []string
		areaId  string
	)

	if len(c.QueryArray("area_id")) == 1 {
		areaIds = getAreaIDs(c)
	}

	if userId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.MissingUserId, c))
		return
	}

	// 判断是否超过流量限额
	ok, err := checkUserTotalFlow(c.Request.Context(), userId)
	if err != nil {
		log.Error("check user total flow error: ", cid)
	}
	if !ok {
		c.JSON(http.StatusOK, respErrorCode(errors.OutTotalFlow, c))
		return
	}

	hash, err := cidutil.CIDToHash(cid)
	if err != nil {
		log.Error("Invalid asset CID: ", cid)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// 如果用户指定了区域，则先判断区域是否存在
	if len(areaIds) > 0 {
		exist, err := dao.CheckUserAssetIsInAreaID(c.Request.Context(), userId, hash, areaIds[0])
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			} else {
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			}
			return
		}
		if !exist {
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
			return
		}
		areaId, err = dao.GetOneAreaIDByAreaID(c.Request.Context(), userId, hash, areaId)
		if err != nil {
			log.Errorf("get one areaid error:%v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		}
	} else {
		// 获取用户文件所有的区域
		areaIDs, err := dao.GetUserAssetAreaIDs(c.Request.Context(), hash, userId)
		if err != nil {
			log.Errorf("get user assest areaids error:%w", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		// 获取用户的访问的ip
		ip, err := GetIPFromRequest(c.Request)
		if err != nil {
			log.Errorf("get user's ip of request error:%w", err)
			// c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			// return
			areaId = areaIDs[0]
		} else {
			areaId, err = GetNearestAreaID(c.Request.Context(), ip, areaIDs)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
				return
			}
		}

	}
	// 获取文件信息
	userAsset, err := dao.GetUserAssetDetail(c.Request.Context(), hash, userId)
	if err != nil {
		log.Error("Failed to get user asset: ", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	user, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		log.Errorf("Failed to get user: %w", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	dao.AddVisitCount(c.Request.Context(), hash, userId)

	n, _ := dao.GetVisitCount(c.Request.Context(), hash, userId)
	if !user.EnableVIP && n > maxCountOfVisitShareLink {
		c.JSON(http.StatusOK, respErrorCode(errors.AssetVisitOutOfLimit, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	var ret []string
	if userAsset.Password != "" {
		urls, err := schedulerClient.ShareEncryptedAsset(c.Request.Context(), userId, cid, userAsset.Password, time.Now().Add(time.Hour*2))
		if err != nil {
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
				return
			}
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		ret = urls
	} else {
		urls, err := schedulerClient.ShareAssets(c.Request.Context(), userId, []string{cid}, time.Now().Add(2*time.Hour))
		if err != nil {
			if webErr, ok := err.(*api.ErrWeb); ok {
				c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
				return
			}
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		ret = urls[cid]
	}

	for i := range ret {
		ret[i] = fmt.Sprintf("%s&filename=%s", ret[i], userAsset.AssetName)
	}

	// 成功的时候，下载量+1
	oprds.GetClient().IncrAssetHourDownload(c.Request.Context(), hash)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"asset_cid": cid,
		"size":      userAsset.TotalSize,
		"url":       ret,
		"redirect":  false,
	}))
}

func ShareLinkInfoHandler(c *gin.Context) {
	username := c.Query("username")
	cid := c.Query("cid")
	sb := squirrel.Select("*").Where("cid = ?", cid).Where("username = ?", username)
	link, err := dao.GetLink(c.Request.Context(), sb)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("GetLink error %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	duration := link.ExpireAt.Sub(link.UpdatedAt)

	days := math.Round(duration.Hours() / 24)

	c.JSON(http.StatusOK, respJSON(gin.H{"link": link, "expire_days": days}))
}

type ShareLinkUpdateReq struct {
	ID        int64  `json:"id"`
	ShortPass string `json:"short_pass"`
	ExpireAt  int64  `json:"expire_at"`
}

func ShareLinkUpdateHandler(c *gin.Context) {
	var req ShareLinkUpdateReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	link, err := dao.GetLink(c.Request.Context(), squirrel.Select("*").Where("id = ?", req.ID))
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("GetLink error %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	if link.UserName != userId {
		log.Errorf("user not match")
		c.JSON(http.StatusOK, respErrorCode(errors.LinkUserNotMatch, c))
		return
	}

	if req.ExpireAt > 0 {
		if time.Now().Unix() > req.ExpireAt {
			log.Errorf("file expired")
			c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
			return
		}
		link.ExpireAt = time.Unix(req.ExpireAt, 0)
		link.UpdatedAt = time.Now() // 只有更新过期时间，才更新updated_at, 防止expire_at - updated_at < 0
	}

	// if req.ShortPass != "" && req.ShortPass != link.ShortPass {
	link.ShortPass = req.ShortPass
	// }

	if err := dao.UpdateLinkPassAndExpiration(c.Request.Context(), link); err != nil {
		log.Errorf("UpdateLink error %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// ShareLinkHandler 获取分享链接
// @Summary 获取分享链接
// @Description 获取分享链接
// @Tags storage
// @Param username query string true "用户id"
// @Param url query string true "url"
// @Param cid query string true "文件cid"
// @Success 200 {object} JsonObject "{url: ""}"
// @Router /api/v1/storage/get_link [get]
func ShareLinkHandler(c *gin.Context) {
	username := c.Query("username")
	cid := c.Query("cid")
	// url := c.Query("url")
	sb := squirrel.Select("*").Where("cid = ?", cid).Where("username = ?", username)
	link, err := dao.GetLink(c.Request.Context(), sb)
	if err != nil {
		log.Errorf("database getLink: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// areaId := getAreaID(c)
	// if cid == "" || url == "" {
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 	return
	// }

	// hash, err := cidutil.CIDToHash(cid)
	// if err != nil {
	// 	log.Errorf("cidToHash: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 	return
	// }

	// asset, err := dao.GetUserAsset(c.Request.Context(), hash, username)
	// if err != nil {
	// 	log.Errorf("database getUserAsset: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 	return
	// }

	// signature := c.Query("signature")
	// if signature != "" {
	// 	fmt.Println("signature:", signature)
	// 	fmt.Println("username:", username)
	// 	nonce := dao.RedisCache.Get(c.Request.Context(), fmt.Sprintf(FilePassNonceVerifyKey, username)).Val()
	// 	if nonce == "" {
	// 		log.Errorf("nonce not found")
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// 	fmt.Println("nonce:", nonce)
	// 	addr, err := rsa.VerifyAddrSign(nonce, signature)
	// 	if err != nil {
	// 		log.Errorf("VerifyAddrSign: %v", err)
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// 	if !strings.EqualFold(addr, username) {
	// 		log.Errorf("addr not match")
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// }

	// access_pass := c.Query("access_pass")
	// if signature != "" && access_pass == "" {
	// 	access_pass = genRandomStr(6)
	// }

	// // if access_pass != "" {
	// // 	asset.ShortPass = access_pass
	// // }

	// expireTime, err := strconv.Atoi(c.Query("expire_time"))
	// if err != nil {
	// 	log.Errorf("expire_time invalid")
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 	return
	// }

	// var expireAt time.Time
	// if expireTime > 0 {
	// 	if time.Now().Unix() > int64(expireTime) {
	// 		log.Errorf("file expired")
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// 	expireAt = time.Unix(int64(expireTime), 0)
	// }

	// if err := dao.UpdateUserAsset(c.Request.Context(), asset); err != nil {
	// 	log.Errorf("database updateUserAsset: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 	return
	// }

	// var link model.Link
	// link.UserName = username
	// link.Cid = cid
	// link.LongLink = url
	// link.ShortPass = access_pass
	// link.ExpireAt = expireAt
	// shortLink := dao.GetShortLink(c.Request.Context(), url)
	// if shortLink == "" {
	// 	link.ShortLink = "/link?" + "cid=" + cid + "&area_id=" + areaId
	// 	shortLink = link.ShortLink
	// 	err := dao.CreateLink(c.Request.Context(), &link)
	// 	if err != nil {
	// 		log.Errorf("database createLink: %v", err)
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 		return
	// 	}
	// } else {
	// 	if !strings.Contains(shortLink, "&area_id=") {
	// 		shortLink = strings.TrimSuffix(shortLink, "&") + "&area_id=" + areaId
	// 	}
	// }

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"url": link.ShortLink,
	}))

}

// CreateShareLinkHandler 获取分享链接
// @Summary 获取分享链接
// @Description 获取分享链接
// @Tags storage
// @Param username query string true "用户id"
// @Param url query string true "url"
// @Param cid query string true "文件cid"
// @Success 200 {object} JsonObject "{url: ""}"
// @Router /api/v1/storage/create_link [get]
func CreateShareLinkHandler(c *gin.Context) {
	var err error
	username := c.Query("username")
	cid := c.Query("cid")
	u := c.Query("url")

	u, err = url.QueryUnescape(u)
	if err != nil {
		log.Errorf("url decode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if cid == "" || u == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	hash, err := cidutil.CIDToHash(cid)
	if err != nil {
		log.Errorf("cidToHash: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	asset, err := dao.GetUserAsset(c.Request.Context(), hash, username)
	if err != nil {
		log.Errorf("database getUserAsset: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// signature := c.Query("signature")
	// if signature != "" {
	// 	fmt.Println("signature:", signature)
	// 	fmt.Println("username:", username)
	// 	nonce := dao.RedisCache.Get(c.Request.Context(), fmt.Sprintf(FilePassNonceVerifyKey, username)).Val()
	// 	if nonce == "" {
	// 		log.Errorf("nonce not found")
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// 	fmt.Println("nonce:", nonce)
	// 	addr, err := rsa.VerifyAddrSign(nonce, signature)
	// 	if err != nil {
	// 		log.Errorf("VerifyAddrSign: %v", err)
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// 	if !strings.EqualFold(addr, username) {
	// 		log.Errorf("addr not match")
	// 		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	// 		return
	// 	}
	// }

	access_pass := c.Query("access_pass")
	// if access_pass == "" {
	// 	access_pass = genRandomStr(6)
	// }

	expireTime, err := strconv.Atoi(c.Query("expire_time"))
	if err != nil {
		log.Errorf("expire_time invalid")
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	var expireAt time.Time
	if expireTime > 0 {
		if time.Now().Unix() > int64(expireTime) {
			log.Errorf("file expired")
			c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
			return
		}
		expireAt = time.Unix(int64(expireTime), 0)
	}

	if err := dao.UpdateUserAsset(c.Request.Context(), asset); err != nil {
		log.Errorf("database updateUserAsset: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var link model.Link
	link.UserName = username
	link.Cid = cid
	link.LongLink = u
	link.ShortPass = access_pass
	link.ExpireAt = expireAt
	link.CreatedAt = time.Now()
	link.UpdatedAt = time.Now()
	shortLink := dao.GetShortLink(c.Request.Context(), u)
	if shortLink == "" {
		link.ShortLink = "/link?" + "cid=" + cid + "&ts=" + strconv.FormatInt(time.Now().Unix(), 10)
		shortLink = link.ShortLink
		err := dao.CreateLink(c.Request.Context(), &link)
		if err != nil {
			log.Errorf("database createLink: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	} else {
		c.JSON(http.StatusOK, respErrorCode(errors.LinkAlreadyExist, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"url": shortLink,
	}))

}

func ShareNeedPassHandler(c *gin.Context) {
	cid := c.Query("cid")
	username := c.Query("username")

	sb := squirrel.Select("*").Where("cid = ?", cid).Where("username = ?", username)
	lk, err := dao.GetLink(c.Request.Context(), sb)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("Error while getting link: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"NeedPass": lk.ShortPass != "",
	}))
}

type CheckShareReq struct {
	Cid      string `json:"cid"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func CheckShareLinkHandler(c *gin.Context) {
	var req CheckShareReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, respErrorCode(errors.InvalidParams, c))
		return
	}
	sb := squirrel.Select("*").Where("cid = ?", req.Cid).Where("username = ?", req.Username)

	link, err := dao.GetLink(c.Request.Context(), sb)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("Error while getting link: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	if link.ExpireAt.Before(time.Now()) {
		c.JSON(http.StatusOK, respErrorCode(errors.ShareLinkExpired, c))
		return
	}

	if link.ShortPass != "" && req.Password == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.ShareLinkPassRequired, c))
		return
	}

	if link.ShortPass != req.Password {
		c.JSON(http.StatusOK, respErrorCode(errors.ShareLinkPassIncorrect, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))

}

const (
	charset                = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	FilePassNonceVerifyKey = "TITAN::FILEPASS_NONCE_VERIFY_%s"
)

func genRandomStr(length int64) string {
	rand.Seed(time.Now().UnixNano())
	randomStr := make([]byte, length)
	for i := range randomStr {
		randomStr[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomStr)
}

func ShareBeforeHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	key := fmt.Sprintf(FilePassNonceVerifyKey, userId)
	nonce := rsa.EncryptPassWithSalt(key + time.Now().String())

	_, err := dao.RedisCache.SetEx(c.Request.Context(), key, nonce, 5*time.Minute).Result()
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"nonce": nonce,
	}))
}

func GetShareLinkHandler(c *gin.Context) {
	cid := c.Query("cid")
	if cid == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	link := dao.GetLongLink(c.Request.Context(), cid)
	if link == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	// 解码 URL
	decodedLink, err := url.QueryUnescape(link)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode URL"})
		return
	}

	c.Redirect(http.StatusMovedPermanently, decodedLink)

}

func UpdateShareStatusHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	cid := c.Query("cid")
	// 获取文件hash
	hash, err := storage.CIDToHash(cid)
	if err != nil {
		log.Errorf("CreateAssetHandler storage.CIDToHash() error: %+v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	err = dao.UpdateAssetShareStatus(c.Request.Context(), hash, userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

type AccessOverview struct {
	AssetRecord      *AssetRecord
	UserAssetDetail  *dao.UserAssetDetail
	VisitCount       int64
	RemainVisitCount int64
	FilcoinCount     int64
}

func GetAssetListHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	// createAssetRsp, err := schedulerClient.ListAssets(c.Request.Context(), userId, pageSize, (page-1)*pageSize, groupId)
	// if err != nil {
	// 	if webErr, ok := err.(*api.ErrWeb); ok {
	// 		c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
	// 		return
	// 	}

	// 	log.Errorf("api ListAssets: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 	return
	// }
	createAssetRsp, err := listAssets(c.Request.Context(), userId, page, pageSize, groupId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var list []*AccessOverview
	for _, asset := range createAssetRsp.AssetOverviews {
		filReplicas, err := dao.CountFilStorage(c.Request.Context(), asset.AssetRecord.CID)
		if err != nil {
			log.Errorf("count fil storage: %v", err)
			continue
		}

		list = append(list, &AccessOverview{
			AssetRecord:      asset.AssetRecord,
			UserAssetDetail:  asset.UserAssetDetail,
			VisitCount:       asset.VisitCount,
			RemainVisitCount: asset.RemainVisitCount,
			FilcoinCount:     filReplicas,
		})
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": createAssetRsp.Total,
	}))
}

func GetAssetAllListHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	groupId, _ := strconv.Atoi(c.Query("group_id"))

	var total int64
	page, size := 1, 100
	var listRsp []*AssetOverview
loop:
	createAssetRsp, err := listAssets(c.Request.Context(), userId, size, size, groupId)
	if err != nil {
		log.Errorf("api ListAssets: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	listRsp = append(listRsp, createAssetRsp.AssetOverviews...)
	total += int64(len(createAssetRsp.AssetOverviews))
	page++
	if total < createAssetRsp.Total {
		goto loop
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  listRsp,
		"total": createAssetRsp.Total,
	}))
}

func GetAssetStatusHandler(c *gin.Context) {
	userId := c.Query("username")
	cid := c.Query("cid")

	statusRsp, err := getAssetStatus(c.Request.Context(), userId, cid)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}
		log.Errorf("api GetAssetStatus: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"data": statusRsp,
	}))
}

func GetOpenAssetStatusHandler(c *gin.Context) {
	userId := c.Query("username")
	cid := c.Query("cid")

	hash, err := storage.CIDToHash(cid)
	if err != nil {
		log.Errorf("Decode CID error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	resp := &types.AssetStatus{
		IsExist:           true,
		IsExpiration:      false,
		IsVisitOutOfLimit: false,
	}

	asset, err := dao.GetUserAsset(c.Request.Context(), hash, userId)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("Get Asset error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if err == sql.ErrNoRows {
		resp.IsExist = false
		c.JSON(http.StatusOK, respJSON(JsonObject{"data": resp}))
		return
	}

	if !asset.Expiration.IsZero() {
		resp.IsExpiration = asset.Expiration.Before(time.Now())
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{"data": resp}))
}

func GetAssetCountHandler(c *gin.Context) {
	var (
		candidateCount, edgeCount = new(atomic.Int64), new(atomic.Int64)
		wg                        = new(sync.WaitGroup)
		deviceExists              = new(sync.Map)
		// deviceExists              = make(map[string]int)
	)

	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	// 获取用户的调度器id信息
	areaHashs, err := dao.GetHashAreaIDList(c.Request.Context(), userId)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	for areaId, hashs := range areaHashs {
		wg.Add(1)
		go func(areaId string, hashs []string) {
			defer wg.Done()

			schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
			if err != nil {
				log.Errorf("get scheduler client error: %v", err)
				return
			}
			for _, hash := range hashs {
				tmpcid, _ := storage.HashToCID(hash)
				assetRsp, err := schedulerClient.GetAssetRecord(c.Request.Context(), tmpcid)
				if err != nil {
					log.Errorf("api GetAssetRecord: %v", err)
					continue
				}
				if len(assetRsp.ReplicaInfos) > 0 {
					for _, rep := range assetRsp.ReplicaInfos {
						// if _, ok := deviceExists[rep.NodeID]; ok {
						// 	continue
						// }
						// deviceExists[rep.NodeID] = 1
						_, ok := deviceExists.LoadOrStore(rep.NodeID, 1)
						if ok {
							continue
						}

						switch rep.IsCandidate {
						case true:
							candidateCount.Add(1)
						default:
							edgeCount.Add(1)
						}
					}
				}
			}
		}(areaId, hashs)
	}
	wg.Wait()

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"area_count":      len(areaHashs),
		"candidate_count": candidateCount.Load(),
		"edge_count":      edgeCount.Load(),
	}))
}

// GetAssetDetailHandler 获取文件详情

func GetAssetDetailHandler(c *gin.Context) {
	uid := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	resp := new(types.AssetRecord)

	// 获取文件hash
	hash, err := storage.CIDToHash(cid)
	if err != nil {
		log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	info, err := dao.GetUserAsset(c.Request.Context(), hash, uid)
	switch err {
	case sql.ErrNoRows:
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	case nil:
	default:
		log.Errorf("get user asset info error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// 获取调度器区域
	areaIds, err := dao.GetAreaIDsByHash(c.Request.Context(), hash)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	for _, v := range areaIds {
		schedulerClient, err := getSchedulerClient(c.Request.Context(), v)
		if err != nil {
			log.Errorf("getSchedulerClient: %v", err)
			continue
		}
		record, err := schedulerClient.GetAssetRecord(c.Request.Context(), cid)
		if err != nil {
			log.Errorf("api GetAssetRecord: %v", err)
			continue
		}
		resp.ReplicaInfos = append(resp.ReplicaInfos, record.ReplicaInfos...)
	}

	cityMap := make(map[string]struct{})

	var deviceIds []string
	for _, replicas := range resp.ReplicaInfos {
		if replicas.Status != 3 {
			continue
		}
		deviceIds = append(deviceIds, replicas.NodeID)
	}

	deviceInfos, e := dao.GetDeviceInfoListByIds(c.Request.Context(), deviceIds)
	if err != nil {
		log.Errorf("GetAssetList err: %v", e)
	}

	for _, nodeInfo := range deviceInfos {
		if _, ok := cityMap[nodeInfo.IpCity]; ok {
			continue
		}
		cityMap[nodeInfo.IpCity] = struct{}{}
	}

	mapList := dao.GenerateDeviceMapInfo(deviceInfos, lang)

	filReplicas, err := dao.CountFilStorage(c.Request.Context(), cid)
	if err != nil {
		log.Errorf("count fil storage: %v", err)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"cid":               cid,
		"cid_name":          info.AssetName,
		"ReplicaInfo_count": len(deviceIds),
		"area_count":        len(cityMap),
		"titan_count":       len(deviceIds),
		"fileCoin_count":    filReplicas,
		"list":              mapList,
		"total":             len(mapList),
	}))
}

func GetLocationHandler(c *gin.Context) {
	var resp = new(types.ListReplicaRsp)
	//userId := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	limit := pageSize
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 获取文件hash
	hash, err := storage.CIDToHash(cid)
	if err != nil {
		log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// 获取调度器区域
	areaIds, err := dao.GetAreaIDsByHash(c.Request.Context(), hash)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	for _, v := range areaIds {
		schedulerClient, err := getSchedulerClient(c.Request.Context(), v)
		if err != nil {
			log.Errorf("getSchedulerClient: %v", err)
			continue
		}
		record, err := schedulerClient.GetAssetRecord(c.Request.Context(), cid)
		if err != nil {
			log.Errorf("api GetAssetRecord: %v", err)
			continue
		}
		resp.Total += len(record.ReplicaInfos)
		resp.ReplicaInfos = append(resp.ReplicaInfos, record.ReplicaInfos...)
	}
	var deviceIds []string
	if len(resp.ReplicaInfos) > 0 {
		for _, rep := range resp.ReplicaInfos {
			if rep.Status != 3 {
				continue
			}
			deviceIds = append(deviceIds, rep.NodeID)
		}
	}

	type DeviceInfoRes struct {
		DeviceId   string
		IpLocation string
		Status     string
	}

	var assetInfos []*DeviceInfoRes
	if len(deviceIds) > 0 {
		assetList, err := dao.GetDeviceInfoListByIds(c.Request.Context(), deviceIds)
		if err != nil {
			log.Errorf("GetAssetList err: %v", err)
		}
		for _, nodeInfo := range assetList {
			loc, lErr := dao.GetCacheLocation(c.Request.Context(), nodeInfo.ExternalIp, lang)
			if lErr == nil && loc != nil {
				nodeInfo.Location = *loc
			}
			assetInfos = append(assetInfos, &DeviceInfoRes{
				DeviceId:   nodeInfo.DeviceID,
				IpLocation: dao.ContactIPLocation(nodeInfo.Location, lang),
				Status:     nodeInfo.DeviceStatus,
			})
		}
	}
	nodeList := make([]*DeviceInfoRes, 0)
	if offset < resp.Total {
		if offset+limit >= resp.Total {
			nodeList = assetInfos[offset:]
		} else {
			nodeList = assetInfos[offset : offset+limit]
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total":     resp.Total,
		"node_list": nodeList,
	}))
}

// GetMapByCidHandler 获取cid map
// @Summary 获取cid map
// @Description 获取cid map
// @Tags storage
// @Param cid query string true "文件cid"
// @Success 200 {object} JsonObject "{url: ""}"
// @Router /api/v1/storage/get_map_cid [get]
func GetMapByCidHandler(c *gin.Context) {
	//userId := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	areaId := getAreaID(c)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	assetRsp, err := schedulerClient.GetAssetRecord(c.Request.Context(), cid)
	if err != nil {
		log.Errorf("api GetAssetRecord: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var deviceIds []string
	if len(assetRsp.ReplicaInfos) > 0 {
		for _, rep := range assetRsp.ReplicaInfos {
			if rep.Status == 3 {
				deviceIds = append(deviceIds, rep.NodeID)
			}
		}
	}

	deviceInfos, e := dao.GetDeviceInfoListByIds(c.Request.Context(), deviceIds)
	if err != nil {
		log.Errorf("GetAssetList err: %v", e)
	}

	mapList := dao.GenerateDeviceMapInfo(deviceInfos, lang)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  mapList,
		"total": len(mapList),
	}))
}

// GetAssetInfoHandler 获取文件信息
// @Summary 上传文件
// @Description 上传文件
// @Security ApiKeyAuth
// @Tags storage
// @Param area_id query string false "节点区域"
// @Param cid query string true "文件cid"
// @Success 200 {object} JsonObject "{{list:[],total:0}}"
// @Router /api/v1/storage/get_asset_info [get]
func GetAssetInfoHandler(c *gin.Context) {
	//userId := c.Query("user_id")
	cid := c.Query("cid")
	areaId := getAreaID(c)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	assetRsp, err := schedulerClient.GetAssetRecord(c.Request.Context(), cid)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}

		log.Errorf("api GetAssetRecord: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var deviceIds []string
	if len(assetRsp.ReplicaInfos) > 0 {
		for _, rep := range assetRsp.ReplicaInfos {
			if rep.Status == 3 {
				deviceIds = append(deviceIds, rep.NodeID)
			}
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":   deviceIds,
		"record": assetRsp,
		"total":  len(deviceIds),
	}))
}

// GetKeyListHandler 获取key列表
// @Summary 获取key列表
// @Description 获取key列表
// @Security ApiKeyAuth
// @Tags storage
// @Success 200 {object} JsonObject "{list:[{name:"",key:"",secret:"",time:""}]}"
// @Router /api/v1/storage/get_keys [get]
func GetKeyListHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	info, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}
	var out []map[string]interface{}
	if len(info.ApiKeys) > 0 {
		keyResp, err := storage.DecodeAPIKeySecrets(info.ApiKeys)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		for k, v := range keyResp {
			item := make(map[string]interface{})
			item["name"] = k
			item["key"] = v.APIKey
			item["secret"] = v.APISecret
			item["time"] = v.CreatedTime
			out = append(out, item)
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list": out,
	}))
}

func GetRetrievalListHandler(c *gin.Context) {
	//nodeId := c.Query("device_id")
	//pageSize, _ := strconv.Atoi(c.Query("page_size"))
	//page, _ := strconv.Atoi(c.Query("page"))
	//
	//if nodeId == "" {
	//	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
	//	return
	//}
	//
	//deviceInfo, err := dao.GetDeviceInfoByID(c.Request.Context(), nodeId)
	//if err != nil {
	//	c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
	//	return
	//}
	//
	//if deviceInfo == nil {
	//	c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
	//	return
	//}

	//schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.IpLocation)
	//if err != nil {
	//	c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
	//	return
	//}
	//resp, err := schedulerClient.GetRetrieveEventRecords(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	//if err != nil {
	//	log.Errorf("api GetRetrieveEventRecords: %v", err)
	//	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	//	return
	//}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  nil,
		"total": 0,
	}))
}

func toValidationEvent(in types.ValidationResultInfo) *model.ValidationEvent {
	return &model.ValidationEvent{
		DeviceID:        in.NodeID,
		ValidatorID:     in.ValidatorID,
		Status:          int32(in.Status),
		Blocks:          in.BlockNumber,
		Time:            in.StartTime,
		Duration:        in.Duration,
		UpstreamTraffic: formatter.ToFixed(float64(in.Duration)*in.Bandwidth, 2),
	}
}

func GetCacheHourHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	start := c.Query("from")
	end := c.Query("to")

	if deviceID == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	dataHour := dao.QueryCacheHour(deviceID, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": dataHour,
	}))
}

func GetCacheDaysHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	start := c.Query("from")
	end := c.Query("to")

	if deviceID == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	dataDaily := dao.QueryCacheDaily(deviceID, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": dataDaily,
	}))
}

func GetAPIKeyPermsHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	keyName := c.Query("key_name")

	var perms []string

	info, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}
	if len(info.ApiKeys) > 0 {
		keyMap, err := storage.DecodeAPIKeys(info.ApiKeys)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		key, ok := keyMap[keyName]
		if !ok {
			c.JSON(http.StatusOK, respErrorCode(int(terrors.APPKeyNotFound), c))
			return
		}
		payload, err := storage.AuthVerify(key.APIKey)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		for _, v := range payload.AccessControlList {
			perms = append(perms, v)
		}
	} else {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"perms": perms,
	}))
}

// CreateGroupHandler 创建文件夹
// @Summary 创建文件夹
// @Description 创建文件夹
// @Security ApiKeyAuth
// @Tags storage
// @Param name query string true "name"
// @Param parent query int true "父级id"
// @Success 200 {object} JsonObject "{group:{}}"
// @Router /api/v1/storage/create_group [get]
func CreateGroupHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	name := c.Query("name")
	parent, _ := strconv.Atoi(c.Query("parent"))
	group, err := dao.CreateAssetGroup(c.Request.Context(), userId, name, parent)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"group": group,
	}))
}

// GetGroupsHandler 获取文件夹列表
// @Summary 获取文件夹列表
// @Description 获取文件夹列表
// @Security ApiKeyAuth
// @Tags storage
// @Param parent query int true "父级id"
// @Param page_size query int true "page_size"
// @Param page query int true "page"
// @Success 200 {object} JsonObject "{list:{},total:0}"
// @Router /api/v1/storage/get_groups [get]
func GetGroupsHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	parent, _ := strconv.Atoi(c.Query("parent"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}

	rsp, err := dao.ListAssetGroupForUser(c.Request.Context(), userId, parent, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  rsp.AssetGroups,
		"total": rsp.Total,
	}))
}

type AssetOrGroup struct {
	AssetOverview *AccessOverview
	Group         interface{}
}

// GetAssetGroupListHandler 获取文件夹列表信息，包含其中的文件信息
// @Summary 获取文件夹列表信息，包含其中的文件信息
// @Description 获取文件夹列表信息，包含其中的文件信息
// @Security ApiKeyAuth
// @Tags storage
// @Param parent query int true "父级id"
// @Param page_size query int true "page_size"
// @Param page query int true "page"
// @Success 200 {object} AssetOrGroup "{list:{},total:0}"
// @Router /api/v1/storage/get_asset_group_list [get]
func GetAssetGroupListHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	lan := c.Request.Header.Get("Lang")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	parentId, _ := strconv.Atoi(c.Query("parent"))

	assetSummary, err := listAssetSummary(c.Request.Context(), userId, parentId, page, pageSize)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}
		log.Errorf("api ListAssetSummary: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	var list []*AssetOrGroup
	for _, assetGroup := range assetSummary.List {
		if assetGroup.AssetGroup != nil {
			list = append(list, &AssetOrGroup{Group: assetGroup.AssetGroup})
			continue
		}

		asset := assetGroup.AssetOverview
		filReplicas, err := dao.CountFilStorage(c.Request.Context(), asset.AssetRecord.CID)
		if err != nil {
			log.Errorf("count fil storage: %v", err)
			continue
		}

		ao := &AccessOverview{
			AssetRecord:      asset.AssetRecord,
			UserAssetDetail:  asset.UserAssetDetail,
			VisitCount:       asset.VisitCount,
			RemainVisitCount: asset.RemainVisitCount,
			FilcoinCount:     filReplicas,
		}
		if ao.UserAssetDetail != nil {
			aids := operateAreaIDs(c.Request.Context(), ao.UserAssetDetail.AreaIDs)
			ao.UserAssetDetail.AreaIDs = aids
			ao.UserAssetDetail.AreaIDMaps = operateAreaMaps(c.Request.Context(), aids, lan)
		}

		list = append(list, &AssetOrGroup{AssetOverview: ao})
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": assetSummary.Total,
	}))
}

// DeleteGroupHandler 删除文件组
func DeleteGroupHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	uid := claims[identityKey].(string)
	gid, _ := strconv.Atoi(c.Query("group_id"))

	if gid == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// 数据库将最外层的文件组删除
	// err := dao.DeleteAssetGroupAndUpdateSize(c.Request.Context(), uid, gid)
	// if err != nil {
	// 	log.Errorf("delete outer asset group error: %v", err)
	// 	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	// 	return
	// }
	// 将用户对应的文件组id塞到asynq中去处理
	// opasynq.DefaultCli.EnqueueAssetGroupID(c.Request.Context(), opasynq.AssetGroupPayload{UserID: uid, GroupID: []int64{int64(gid)}})

	err := dao.DeleteAssetGroup(c.Request.Context(), uid, gid)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// RenameGroupHandler 文件重命名
// @Summary 文件重命名
// @Description 文件重命名
// @Security ApiKeyAuth
// @Tags storage
// @Pamam req body RenameAssetReq true "文件重命名请求参数"
// @Success 200 {object} JsonObject "{"msg":"success"}"
// @Router /api/v1/storage/rename_group [post]
func RenameGroupHandler(c *gin.Context) {
	var req RenameAssetReq

	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Errorf("params error:%w", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if req.AssetCID != "" {
		// 获取文件hash
		hash, err := storage.CIDToHash(req.AssetCID)
		if err != nil {
			log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		err = dao.UpdateAssetName(c.Request.Context(), req.NewName, userId, hash)
		if err != nil {
			log.Errorf("update asset name error: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	} else {
		err = dao.UpdateAssetGroupName(c.Request.Context(), userId, req.NewName, req.GroupID)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// RenameAssetHandler 文件重命名
// @Summary 文件重命名
// @Description 文件重命名
// @Security ApiKeyAuth
// @Tags storage
// @Pamam req body RenameAssetReq true "文件重命名请求参数"
// @Success 200 {object} JsonObject "{"msg":"success"}"
// @Router /api/v1/storage/rename_asset [post]
func RenameAssetHandler(c *gin.Context) {
	var req RenameAssetReq

	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Errorf("params error:%w", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// 获取文件hash
	hash, err := storage.CIDToHash(req.AssetCID)
	if err != nil {
		log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	err = dao.UpdateAssetName(c.Request.Context(), req.NewName, userId, hash)
	if err != nil {
		log.Errorf("update asset name error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func MoveGroupToGroupHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	targetGroupId, _ := strconv.Atoi(c.Query("target_group_id"))

	err := dao.MoveAssetGroup(c.Request.Context(), userId, groupId, targetGroupId)
	if err != nil {
		log.Errorf("api MoveAssetGroup: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func MoveAssetToGroupHandler(c *gin.Context) {
	// userId := c.Query("user_id")
	claims := jwt.ExtractClaims(c)
	userId := claims[identityKey].(string)
	assetCid := c.Query("asset_cid")
	groupId, _ := strconv.Atoi(c.Query("group_id"))

	// 获取文件hash
	hash, err := storage.CIDToHash(assetCid)
	if err != nil {
		log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	err = dao.UpdateAssetGroup(c.Request.Context(), userId, hash, groupId)
	if err != nil {
		log.Errorf("UpdateAssetGroup error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

// GetSchedulerAreaIDs 获取调度器的 area id 列表
// @Summary 获取调度器的 area id 列表
// @Description 获取调度器的 area id 列表
// @Tags storage
// @Success 200 {object} JsonObject "{list:[]}"
// @Router /api/v1/storage/get_area_id [get]
func GetSchedulerAreaIDs(c *gin.Context) {
	var (
		keys []string
		err  error
	)

	lan := c.Request.Header.Get("Lang")
	cid := strings.TrimSpace(c.Query("cid"))
	uid := strings.TrimSpace(c.Query("user_id"))

	if cid != "" {
		hash, err := storage.CIDToHash(cid)
		if err != nil {
			log.Errorf("CreateAssetHandler CIDToHash error: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		aids, err := dao.UserAssetAreaIDs(c.Request.Context(), uid, hash)
		if err != nil {
			log.Errorf("get area_ids error: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
		keys = operateAreaIDs(c.Request.Context(), aids)
	} else {
		keys, _, err = GetAndStoreAreaIDs()
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}

	maps := operateAreaMaps(c.Request.Context(), keys, lan)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":      keys,
		"area_maps": maps,
	}))
}

// MoveNode 将调度器节点进行迁移
// @Summary 将调度器节点进行迁移
func MoveNode(c *gin.Context) {
	var req MoveNodeReq

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	// 将node节点从from area移出
	fscli, err := getSchedulerClient(c.Request.Context(), req.FromAreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	info, err := fscli.MigrateNodeOut(c.Request.Context(), req.NodeID)
	if err != nil {
		log.Errorf("exec MigrateNodeOut error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	tscli, err := getSchedulerClient(c.Request.Context(), req.ToAreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	err = tscli.MigrateNodeIn(c.Request.Context(), info)
	if err != nil {
		log.Errorf("exec MigrateNodeIn error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	err = fscli.CleanupNode(c.Request.Context(), req.NodeID, info.Key)
	if err != nil {
		log.Errorf("exec CleanNode error: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "success",
	})
}

// GetMonitor 获取在线的数据
func GetMonitor(c *gin.Context) {
	online, err := dao.GetOnlineNodes(c.Request.Context())
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	country, err := dao.GetCountryCount(c.Request.Context())
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	fil, err := GetFILPrice(c.Request.Context())
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(gin.H{
		"online":   online,
		"country":  country,
		"filecoin": "100+",
		"deposit":  1200 * 5.7 * fil * 1024,
	}))
}
