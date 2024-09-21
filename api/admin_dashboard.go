package api

import (
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/statistics"
	"net/http"
	"strconv"
)

func GetTotalStatsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	areaId := c.Query("area_id")
	userStats, err := dao.GetTotalUserStats(ctx)
	if err != nil {
		log.Errorf("get total user stats: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	nodeStats, err := dao.GetTotalNodeStats(ctx, areaId)
	if err != nil {
		log.Errorf("get total node stats: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	nodeStats.OfflineIPs = nodeStats.TotalIPs - nodeStats.OnlineIPs

	assetStats := &model.TotalAssetStats{}

	out := struct {
		*model.TotalNodeStats
		*model.TotalUserStats
		*model.TotalAssetStats
	}{
		nodeStats, userStats, assetStats,
	}

	c.JSON(http.StatusOK, respJSON(out))
	return
}

func GetNodeIPChangedRecordsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Query("node_id")
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("page_size"), 10, 64)

	option := dao.QueryOption{
		Page:     int(page),
		PageSize: int(size),
	}

	total, records, err := dao.GetNodeIPChangedRecords(ctx, id, option)
	if err != nil {
		log.Errorf("get ip changed history: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total":   total,
		"records": records,
	}))
}

func GetAssetRecordsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	cid := c.Query("cid")

	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	option := dao.QueryOption{
		Page:     page,
		PageSize: pageSize,
	}

	total, records, err := dao.GetAssetsList(ctx, cid, option)
	if err != nil {
		log.Errorf("get assets list: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total":   total,
		"records": records,
	}))
}

func GetSuccessfulReplicasHandler(c *gin.Context) {
	nodeId := c.Query("node_id")
	cid := c.Query("cid")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	//var areaId string
	if nodeId != "" {
		deviceInfo, err := dao.GetDeviceInfoByID(ctx, nodeId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
			return
		}

		schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
			return
		}

		resp, err := schedulerClient.GetSucceededReplicaByNode(ctx, nodeId, limit, offset)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		c.JSON(http.StatusOK, respJSON(resp))
		return
	}

	if cid != "" {
		assetInfo, err := dao.GetAssetByCID(ctx, cid)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		schedulerClient, err := getSchedulerClient(c.Request.Context(), assetInfo.AreaId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
			return
		}

		resp, err := schedulerClient.GetSucceededReplicaByCID(ctx, cid, limit, offset)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		c.JSON(http.StatusOK, respJSON(resp))
		return
	}

	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
}

func GetFailedReplicasHandler(c *gin.Context) {
	nodeId := c.Query("node_id")
	cid := c.Query("cid")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	//var areaId string
	if nodeId != "" {
		deviceInfo, err := dao.GetDeviceInfoByID(ctx, nodeId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
			return
		}

		schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
			return
		}

		resp, err := schedulerClient.GetFailedReplicaByNode(ctx, nodeId, limit, offset)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		c.JSON(http.StatusOK, respJSON(resp))
		return
	}

	if cid != "" {
		assetInfo, err := dao.GetAssetByCID(ctx, cid)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		schedulerClient, err := getSchedulerClient(c.Request.Context(), assetInfo.AreaId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
			return
		}

		resp, err := schedulerClient.GetFailedReplicaByCID(ctx, cid, limit, offset)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}

		c.JSON(http.StatusOK, respJSON(resp))
		return
	}

	c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
}

func GetQualitiesNodesHandler(c *gin.Context) {
	areaId := c.Query("area_id")
	nodeId := c.Query("node_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	ctx := c.Request.Context()

	option := dao.QueryOption{
		Page:     page,
		PageSize: pageSize,
	}

	total, result, err := dao.GetQualitiesNodes(ctx, areaId, nodeId, option)
	if err != nil {
		log.Errorf("get qualities nodes: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total": total,
		"list":  result,
	}))
	return
}

func GetWorkerdNodesHandler(c *gin.Context) {
	nodeId := c.Query("node_id")
	areaId := c.Query("area_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	ctx := c.Request.Context()

	option := dao.QueryOption{
		Page:     page,
		PageSize: pageSize,
	}

	total, result, err := dao.GetWorkerdNodes(ctx, areaId, nodeId, option)
	if err != nil {
		log.Errorf("get workerd nodes: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"total": total,
		"list":  result,
	}))
	return
}

func GetAreasHandler(c *gin.Context) {
	lang := c.GetHeader("Lang")

	type Areas struct {
		AreaId string `json:"area_id"`
		Name   string `json:"name"`
	}

	var out []Areas

	for _, scheduler := range statistics.Schedulers {
		a := Areas{
			AreaId: scheduler.AreaId,
			Name:   scheduler.AreaId,
		}

		aids := operateAreaIDs(c.Request.Context(), []string{scheduler.AreaId})
		names := operateAreaMaps(c.Request.Context(), aids, lang)

		if len(names) > 0 {
			a.Name = names[0].Key
		}

		out = append(out, a)
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"areas": out,
	}))
}

func GetProjectOverviewHandler(c *gin.Context) {
	nodeId := c.Query("node_id")
	areaId := c.Query("area_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	if nodeId != "" {
		deviceInfo, err := dao.GetDeviceInfoByID(ctx, nodeId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
			return
		}

		areaId = deviceInfo.AreaID
	}

	if areaId == "" {
		areaSchMaps.Range(func(key, value any) bool {
			areaId = key.(string)
			return false
		})
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	resp, err := schedulerClient.GetProjectOverviewByNode(ctx, &types.NodeProjectReq{
		NodeID: nodeId,
		Limit:  limit,
		Offset: offset,
	})

	if err != nil {
		log.Errorf("GetProjectOverviewByNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(resp))

}

func GetProjectInfoHandler(c *gin.Context) {
	nodeId := c.Query("node_id")
	projectId := c.Query("project_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	deviceInfo, err := dao.GetDeviceInfoByID(ctx, nodeId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), deviceInfo.AreaID)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	resp, err := schedulerClient.GetProjectReplicasForNode(ctx, &types.NodeProjectReq{
		NodeID:    nodeId,
		ProjectID: projectId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		log.Errorf("GetProjectReplicasForNode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(resp))
}
