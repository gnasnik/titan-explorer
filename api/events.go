package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gnasnik/titan-explorer/pkg/formatter"

	"github.com/Filecoin-Titan/titan/api"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
)

func GetCacheListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	//device, err := dao.GetDeviceInfo(c.Request.Context(), nodeId)
	//if err != nil {
	//	log.Errorf("get device info: %v", err)
	//	c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
	//	return
	//}
	//

	// todo: get scheduler from area id
	resp, err := schedulerApi.GetReplicaEventsForNode(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api GetReplicaEventsForNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.ReplicaEvents,
		"total": resp.Total,
	}))
	return
}

func GetValidationListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	resp, err := schedulerApi.GetValidationResults(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api GetValidationResults: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	var validationEvents []*model.ValidationEvent
	for _, blockInfo := range resp.ValidationResultInfos {
		validationEvents = append(validationEvents, toValidationEvent(blockInfo))
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  validationEvents,
		"total": resp.Total,
	}))
	return
}

func GetAllocateStorageHandler(c *gin.Context) {
	userId := c.Query("user_id")
	if userId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	var userInfo model.User
	userInfo.Username = userId
	_, err := dao.GetUserByUsername(c.Request.Context(), userInfo.Username)
	if err == nil {
		log.Info("GetUserByUsername user exists")
	} else {
		_ = dao.CreateUser(c.Request.Context(), &userInfo)
	}

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	_, err = schedulerClient.AllocateStorage(c.Request.Context(), userId)
	if err != nil {
		log.Errorf("api GetValidationResults: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
	return
}

func GetStorageSizeHandler(c *gin.Context) {
	userId := c.Query("user_id")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	storageSize, err := schedulerClient.GetUserInfo(c.Request.Context(), userId)
	if err != nil {
		log.Errorf("api GetStorageSize: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	peakBandwidth := GetUserInfo(c.Request.Context(), userId)
	if peakBandwidth > storageSize.PeakBandwidth {
		storageSize.PeakBandwidth = peakBandwidth
	} else {
		var expireTime time.Duration
		expireTime = time.Hour
		// update redis data
		_ = SetUserInfo(c.Request.Context(), userId, storageSize.PeakBandwidth, expireTime)
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"PeakBandwidth": storageSize.PeakBandwidth,
		"TotalTraffic":  storageSize.TotalTraffic,
		"TotalSize":     storageSize.TotalSize,
		"UsedSize":      storageSize.UsedSize,
	}))
	return
}

func GetUserVipInfoHandler(c *gin.Context) {
	userId := c.Query("user_id")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	storageSize, err := schedulerClient.GetUserInfo(c.Request.Context(), userId)
	if err != nil {
		log.Errorf("api GetUserInfo: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"vip": storageSize.EnableVIP,
	}))
	return
}

func GetUserAccessTokenHandler(c *gin.Context) {
	UserId := c.Query("user_id")
	areaId := dao.GetAreaID(c.Request.Context(), UserId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	token, err := schedulerClient.GetUserAccessToken(c.Request.Context(), UserId)
	if err != nil {
		log.Errorf("api GetUserAccessToken: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"AccessToken": token,
	}))
}

func CreateAssetHandler(c *gin.Context) {
	userId := c.Query("user_id")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	user, err := dao.GetUserByUsername(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	nodeIPInfos, err := schedulerClient.GetCandidateIPs(c.Request.Context())
	if err != nil {
		log.Warnf("get candidate ips error %s", err.Error())
	}

	var nearestNode string
	if len(nodeIPInfos) > 0 {
		nodeMap := make(map[string]string)
		ips := make([]string, 0, len(nodeIPInfos))
		for _, nodeIPInfo := range nodeIPInfos {
			ips = append(ips, nodeIPInfo.IP)
			nodeMap[nodeIPInfo.IP] = nodeIPInfo.NodeID
		}

		if ip, err := GetUserNearestIP(c.Request.Context(), c.ClientIP(), ips, NewIPCoordinate()); err == nil {
			nearestNode = nodeMap[ip]
		} else {
			log.Warnf("GetUserNearestIP error %s", err.Error())
		}
	}

	log.Debugf("CreateAssetHandler clientIP:%s, areaId:%s, nearestNode:%s\n", c.ClientIP(), areaId, nearestNode)

	var createAssetReq types.CreateAssetReq
	createAssetReq.AssetName = c.Query("asset_name")
	createAssetReq.AssetCID = c.Query("asset_cid")
	createAssetReq.UserID = userId
	createAssetReq.AssetType = c.Query("asset_type")
	createAssetReq.AssetSize = formatter.Str2Int64(c.Query("asset_size"))
	createAssetReq.GroupID, _ = strconv.Atoi(c.Query("group_id"))
	createAssetReq.NodeID = nearestNode

	if err := dao.AddAssets(c.Request.Context(), []*model.Asset{
		{
			UserId:    userId,
			Name:      createAssetReq.AssetName,
			Cid:       createAssetReq.AssetCID,
			Type:      createAssetReq.AssetType,
			NodeID:    createAssetReq.NodeID,
			TotalSize: createAssetReq.AssetSize,
			Event:     -1,
			ProjectId: user.ProjectId,
		},
	}); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	createAssetRsp, err := schedulerClient.CreateAsset(c.Request.Context(), &createAssetReq)
	if err != nil {
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
			return
		}
	}
	if createAssetRsp.AlreadyExists {
		c.JSON(http.StatusOK, respErrorCode(errors.FileExists, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"CandidateAddr": createAssetRsp.UploadURL,
		"Token":         createAssetRsp.Token,
	}))
}

func CreateKeyHandler(c *gin.Context) {
	userId := c.Query("user_id")
	keyName := c.Query("key_name")
	permsStr := c.Query("perms")

	perms := strings.Split(permsStr, ",")
	acl := make([]types.UserAccessControl, 0, len(perms))
	for _, perm := range perms {
		acl = append(acl, types.UserAccessControl(perm))
	}

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	keyStr, err := schedulerClient.CreateAPIKey(c.Request.Context(), userId, keyName, acl)
	if err != nil {
		log.Errorf("api CreateAPIKey: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"key": keyStr,
	}))
}

func DeleteKeyHandler(c *gin.Context) {
	userId := c.Query("user_id")
	keyName := c.Query("key_name")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.DeleteAPIKey(c.Request.Context(), userId, keyName)
	if err != nil {
		log.Errorf("api DeleteAPIKey: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "delete success",
	}))
}

func DeleteAssetHandler(c *gin.Context) {
	UserId := c.Query("user_id")
	cid := c.Query("asset_cid")
	areaId := dao.GetAreaID(c.Request.Context(), UserId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.DeleteAsset(c.Request.Context(), UserId, cid)
	if err != nil {
		log.Errorf("api DeleteAsset: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "delete success",
	}))
}

func ShareAssetsHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("asset_cid")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	urls, err := schedulerClient.ShareAssets(c.Request.Context(), userId, []string{cid})
	if err != nil {
		log.Errorf("api ShareAssets: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	// todo: remove this
	var redirect bool
	if userId == "f1hl7vbopivazgion4ql25opmjnoj2ldfsvm5fuzi" {
		redirect = false
	} else {
		redirect = true
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"asset_cid": cid,
		"url":       urls[cid],
		"redirect":  redirect,
	}))
}

func ShareLinkHandler(c *gin.Context) {
	username := c.Query("username")
	cid := c.Query("cid")
	url := c.Query("url")
	if cid == "" || url == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	var link model.Link
	link.UserName = username
	link.Cid = cid
	link.LongLink = url
	shortLink := dao.GetShortLink(c.Request.Context(), url)
	if shortLink == "" {
		link.ShortLink = "/link?" + "cid=" + cid
		shortLink = link.ShortLink
		err := dao.CreateLink(c.Request.Context(), &link)
		if err != nil {
			log.Errorf("database createLink: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"url": shortLink,
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
	c.Redirect(http.StatusMovedPermanently, link)

}

func UpdateShareStatusHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("cid")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.UpdateShareStatus(c.Request.Context(), userId, cid)
	if err != nil {
		log.Errorf("api UpdateShareStatus: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

type AccessOverview struct {
	AssetRecord      *types.AssetRecord
	UserAssetDetail  *types.UserAssetDetail
	VisitCount       int
	RemainVisitCount int
	FilcoinCount     int64
}

func GetAssetListHandler(c *gin.Context) {
	userId := c.Query("user_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	createAssetRsp, err := schedulerClient.ListAssets(c.Request.Context(), userId, pageSize, (page-1)*pageSize, groupId)
	if err != nil {
		log.Errorf("api ListAssets: %v", err)
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
	userId := c.Query("user_id")
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	var total int
	page, size := 1, 100
	var listRsp []*types.AssetOverview
loop:
	createAssetRsp, err := schedulerClient.ListAssets(c.Request.Context(), userId, size, (page-1)*size, groupId)
	if err != nil {
		log.Errorf("api ListAssets: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	listRsp = append(listRsp, createAssetRsp.AssetOverviews...)
	total += len(createAssetRsp.AssetOverviews)
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
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	statusRsp, err := schedulerClient.GetAssetStatus(c.Request.Context(), userId, cid)
	if err != nil {
		log.Errorf("api GetAssetStatus: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"data": statusRsp,
	}))
}

func GetAssetCountHandler(c *gin.Context) {
	userId := c.Query("user_id")
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize = 100
	page = 1
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	createAssetRsp, err := schedulerClient.ListAssets(c.Request.Context(), userId, pageSize, (page-1)*pageSize, groupId)
	if err != nil {
		log.Errorf("api ListAssets: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	var deviceIds []string
	deviceExists := make(map[string]int)
	var candidateCount int64
	var edgeCount int64
	for _, data := range createAssetRsp.AssetOverviews {
		if len(data.AssetRecord.ReplicaInfos) > 0 {
			for _, rep := range data.AssetRecord.ReplicaInfos {
				if _, ok := deviceExists[rep.NodeID]; ok {
					continue
				}
				deviceExists[rep.NodeID] = 1
				deviceIds = append(deviceIds, rep.NodeID)
				switch rep.IsCandidate {
				case true:
					candidateCount += 1
				default:
					edgeCount += 1
				}
			}
		}
	}
	countArea, e := dao.GetAreaCount(c.Request.Context(), deviceIds)
	if e != nil {
		log.Errorf("GetAssetList err: %v", e)
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"area_count":      countArea,
		"candidate_count": candidateCount,
		"edge_count":      edgeCount,
	}))
}

func GetCarFileCountHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
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

	var deviceIdAll []string
	deviceExists := make(map[string]int)
	if len(assetRsp.ReplicaInfos) > 0 {
		for _, rep := range assetRsp.ReplicaInfos {
			if rep.Status == 3 {
				deviceIdAll = append(deviceIdAll, rep.NodeID)
				continue
			}
		}
	}

	assetListAll, e := dao.GetAssetList(c.Request.Context(), deviceIdAll, lang, dao.QueryOption{})
	if err != nil {
		log.Errorf("GetAssetList err: %v", e)
	}

	for _, nodeInfo := range assetListAll {
		if _, ok := deviceExists[nodeInfo.IpCity]; ok {
			continue
		}
		deviceExists[nodeInfo.IpCity] = 1
	}

	filReplicas, err := dao.CountFilStorage(c.Request.Context(), cid)
	if err != nil {
		log.Errorf("count fil storage: %v", err)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"cid":               assetRsp.CID,
		"cid_name":          "",
		"ReplicaInfo_count": len(deviceIdAll),
		"area_count":        len(deviceExists),
		"titan_count":       len(deviceIdAll),
		"fileCoin_count":    filReplicas,
	}))
}

func GetLocationHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	resp, err := schedulerClient.GetReplicas(c.Request.Context(), cid, 9999, 0)
	if err != nil {
		log.Errorf("api GetReplicas: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	var deviceIds []string
	if len(resp.ReplicaInfos) > 0 {
		for _, rep := range resp.ReplicaInfos {
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
		assetList, err := dao.GetAssetList(c.Request.Context(), deviceIds, lang, dao.QueryOption{PageSize: pageSize, Page: page})
		if err != nil {
			log.Errorf("GetAssetList err: %v", err)
		}
		for _, nodeInfo := range assetList {
			assetInfos = append(assetInfos, &DeviceInfoRes{
				DeviceId:   nodeInfo.DeviceID,
				IpLocation: dao.ContactIPLocation(nodeInfo.Location, lang),
				Status:     nodeInfo.DeviceStatus,
			})
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total":     resp.Total,
		"node_list": assetInfos,
	}))
}

func GetMapByCidHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("cid")
	lang := model.Language(c.GetHeader("Lang"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
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

	assetList, e := dao.GetAssetList(c.Request.Context(), deviceIds, lang, dao.QueryOption{})
	if err != nil {
		log.Errorf("GetAssetList err: %v", e)
	}
	mapList := dao.HandleMapInfo(assetList, lang)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  mapList,
		"total": len(mapList),
	}))
}

func GetAssetInfoHandler(c *gin.Context) {
	userId := c.Query("user_id")
	cid := c.Query("cid")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
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

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  deviceIds,
		"total": len(deviceIds),
	}))
}

func GetKeyListHandler(c *gin.Context) {
	userId := c.Query("user_id")
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	keyResp, err := schedulerClient.GetAPIKeys(c.Request.Context(), userId)
	if err != nil {
		log.Errorf("api GetAPIKeys: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	var out []map[string]interface{}
	for k, v := range keyResp {
		item := make(map[string]interface{})
		item["name"] = k
		item["key"] = v.APIKey
		item["time"] = v.CreatedTime
		out = append(out, item)
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list": out,
	}))
}

func GetRetrievalListHandler(c *gin.Context) {
	nodeId := c.Query("device_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	resp, err := schedulerApi.GetRetrieveEventRecords(c.Request.Context(), nodeId, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api GetRetrieveEventRecords: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  resp.RetrieveEventInfos,
		"total": resp.Total,
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
	dataHour := dao.QueryCacheHour(deviceID, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": dataHour,
	}))
}

func GetCacheDaysHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	start := c.Query("from")
	end := c.Query("to")
	dataDaily := dao.QueryCacheDaily(deviceID, start, end)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"series_data": dataDaily,
	}))
}

func GetAPIKeyPermsHandler(c *gin.Context) {
	userId := c.Query("user_id")
	keyName := c.Query("key_name")

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	perms, err := schedulerClient.GetAPPKeyPermissions(c.Request.Context(), userId, keyName)
	if err != nil {
		log.Errorf("api GetAPPKeyPermissions: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"perms": perms,
	}))
}

func CreateGroupHandler(c *gin.Context) {
	userId := c.Query("user_id")
	name := c.Query("name")
	parent, _ := strconv.Atoi(c.Query("parent"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	group, err := schedulerClient.CreateAssetGroup(c.Request.Context(), userId, name, parent)
	if err != nil {
		log.Errorf("api CreateAssetGroup: %v", err)
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

func GetGroupsHandler(c *gin.Context) {
	userId := c.Query("user_id")
	parent, _ := strconv.Atoi(c.Query("parent"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	rsp, err := schedulerClient.ListAssetGroup(c.Request.Context(), userId, parent, pageSize, (page-1)*pageSize)
	if err != nil {
		log.Errorf("api ListAssetGroup: %v", err)
		if webErr, ok := err.(*api.ErrWeb); ok {
			c.JSON(http.StatusOK, respErrorCode(webErr.Code, c))
		} else {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		}
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

func GetAssetGroupListHandler(c *gin.Context) {
	userId := c.Query("user_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	parentId, _ := strconv.Atoi(c.Query("parent"))
	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	assetSummary, err := schedulerClient.ListAssetSummary(c.Request.Context(), userId, parentId, pageSize, (page-1)*pageSize)
	if err != nil {
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

		list = append(list, &AssetOrGroup{AssetOverview: ao})
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": assetSummary.Total,
	}))
}

func DeleteGroupHandler(c *gin.Context) {
	userId := c.Query("user_id")
	groupId, _ := strconv.Atoi(c.Query("group_id"))

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.DeleteAssetGroup(c.Request.Context(), userId, groupId)
	if err != nil {
		log.Errorf("api DeleteAssetGroup: %v", err)
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

func RenameGroupHandler(c *gin.Context) {
	userId := c.Query("user_id")
	newName := c.Query("new_name")
	groupId, _ := strconv.Atoi(c.Query("group_id"))

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.RenameAssetGroup(c.Request.Context(), userId, newName, groupId)
	if err != nil {
		log.Errorf("api RenameAssetGroup: %v", err)
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

func MoveGroupToGroupHandler(c *gin.Context) {
	userId := c.Query("user_id")
	groupId, _ := strconv.Atoi(c.Query("group_id"))
	targetGroupId, _ := strconv.Atoi(c.Query("target_group_id"))

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.MoveAssetGroup(c.Request.Context(), userId, groupId, targetGroupId)
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
	userId := c.Query("user_id")
	assetCid := c.Query("asset_cid")
	groupId, _ := strconv.Atoi(c.Query("group_id"))

	areaId := dao.GetAreaID(c.Request.Context(), userId)
	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}
	err = schedulerClient.MoveAssetToGroup(c.Request.Context(), userId, assetCid, groupId)
	if err != nil {
		log.Errorf("api MoveAssetToGroup: %v", err)
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
