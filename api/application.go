package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	"net/http"
	"net/mail"
	"strconv"
	"time"
)

// CreateApplicationHandler is a function to create an application for activation code
//
//	@Summary		申请激活码
//	@Tags			申请激活码
//	@Description	GetDeviceInfoHandler is a function to create an application for activation code
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.Application	true	"请求参数"
//	@Success		200		{object}	JsonObject{}
//	@Failure		400		{object}	errors.GenericError
//	@Router			/create_application [post]
func CreateApplicationHandler(c *gin.Context) {
	params := model.Application{}
	if err := c.BindJSON(&params); err != nil {
		log.Errorf("create application: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInvalidParams))
		return
	}

	_, err := mail.ParseAddress(params.Email)
	if err != nil || params.UserID == "" || params.NodeType == 0 {
		c.JSON(http.StatusBadRequest, respError(errors.ErrInvalidParams))
		return
	}

	if params.Amount <= 0 {
		params.Amount = 1
	}

	if params.Amount > 500 {
		c.JSON(http.StatusBadRequest, respError(errors.ErrAmountLimitExceeded))
		return
	}

	params.CreatedAt = time.Now()
	params.UpdatedAt = time.Now()
	params.Status = dao.ApplicationStatusCreated
	params.Ip = utils.GetClientIP(c.Request)
	if err := dao.AddApplication(c.Request.Context(), &params); err != nil {
		log.Errorf("add application: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

// GetApplicationsHandler is a function to get a slice of record(s) from application table in database
//
//	@Summary		获取申请激活码记录
//	@Tags			申请激活码
//	@Description	GetApplicationsHandler is a function to get a slice of record(s) from application table in database
//	@Accept			json
//	@Produce		json
//	@Param			user_id		query		string	true	"用户ID"
//	@Param			page		query		int		false	"分页页码，默认 1"
//	@Param			page_size	query		int		false	"分页页数 默认 50"
//	@Param			order		query		string	false	"排序， Enums(ASC, DESC)"
//	@Param			order_field	query		string	false	"排序字段"
//	@Success		200			{object}	JsonObject{list=[]model.Application}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/get_applications [get]
func GetApplicationsHandler(c *gin.Context) {
	userID := c.Query("user_id")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		UserID:     userID,
		Order:      order,
		OrderField: orderField,
	}
	applications, total, err := dao.GetApplicationsByPage(c.Request.Context(), option)
	if err != nil {
		log.Errorf("get applications: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  applications,
		"total": total,
	}))
}
