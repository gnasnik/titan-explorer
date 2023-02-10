package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"net/http"
	"strconv"
)

// GetCacheListHandler is a function to get a slice of record(s) from CacheEvent table in database
//
//	@Summary		获取设备的缓存列表
//	@Tags			矿工页
//	@Description	GetDeviceInfoHandler is a handler to get a slice of record(s) from CacheEvent table in database
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			page		query		int		false	"分页页码，默认 1"
//	@Param			page_size	query		int		false	"分页页数 默认 50"
//	@Param			order		query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field	query		string	false	"排序字段"
//	@Success		200			{object}	JsonObject{list=[]model.CacheEvent}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_cache_list [get]
func GetCacheListHandler(c *gin.Context) {
	info := &model.CacheEvent{
		DeviceID: c.Query("device_id"),
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		OrderField: orderField,
		Order:      order,
	}

	list, total, err := dao.GetCacheEventsByPage(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get cache events by page: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrNotFound))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

// GetRetrievalListHandler is a function to get a slice of record(s) from RetrievalEvent table in database
//
//	@Summary		获取检索数据列表
//	@Tags			矿工页
//	@Description	GetRetrievalListHandler is a handler to get a slice of record(s) from RetrievalEvent table in database
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			page		query		int		false	"分页页码，默认 1"
//	@Param			page_size	query		int		false	"分页页数 默认 50"
//	@Param			order		query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field	query		string	false	"排序字段"
//	@Success		200			{object}	JsonObject{list=[]model.RetrievalEvent}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_retrieval_list [get]
func GetRetrievalListHandler(c *gin.Context) {
	info := &model.RetrievalEvent{
		DeviceID: c.Query("device_id"),
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	list, total, err := dao.GetRetrievalEventsByPage(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get retrives by page: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrNotFound))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

// GetValidationListHandler is a function to get a slice of record(s) from ValidationEvent table in database
//
//	@Summary		获取验证数据列表
//	@Tags			矿工页
//	@Description	GetValidationListHandler is a handler to get a slice of record(s) from ValidationEvent table in database
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			page		query		int		false	"分页页码，默认 1"
//	@Param			page_size	query		int		false	"分页页数 默认 50"
//	@Param			order		query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field	query		string	false	"排序字段"
//	@Success		200			{object}	JsonObject{list=[]model.ValidationEvent}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_validation_list [get]
func GetValidationListHandler(c *gin.Context) {
	info := &model.ValidationEvent{
		DeviceID: c.Query("device_id"),
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")

	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	list, total, err := dao.GetValidationEventsByPage(c.Request.Context(), info, option)
	if err != nil {
		log.Errorf("get validations by page: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrNotFound))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}
