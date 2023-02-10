package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	"net/http"
	"time"
)

// DeviceBindingHandler is a function to binding device
//
//	@Summary		设备绑定
//	@Tags			控制台
//	@Description	DeviceBindingHandler is a function to create an application for activation code
//	@Accept			json
//	@Produce		json
//	@Param			device_id	query		string	true	"设备ID"
//	@Param			user_id		query		string	true	"用户ID"
//	@Success		200			{object}	JsonObject{}
//	@Failure		400			{object}	errors.GenericError
//	@Router			/device_binding [get]
func DeviceBindingHandler(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	deviceInfo.UserID = c.Query("user_id")
	deviceInfo.BindStatus = "binding"

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	if old != nil && old.UserID != "" {
		c.JSON(http.StatusBadRequest, respError(errors.ErrDeviceExists))
		return
	}
	var timeWeb = "0000-00-00 00:00:00"
	timeString, _ := time.Parse(utils.TimeFormatYMDHmS, timeWeb)
	if old != nil && old.BoundAt == timeString {
		deviceInfo.BoundAt = time.Now()
	}
	err = dao.UpdateUserDeviceInfo(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func DeviceUnBindingHandler(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	deviceInfo.UserID = c.Query("user_id")
	deviceInfo.BindStatus = "unbinding"

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	if old == nil {
		c.JSON(http.StatusBadRequest, respError(errors.ErrDeviceNotExists))
		return
	}

	if old.UserID != deviceInfo.UserID {
		c.JSON(http.StatusBadRequest, respError(errors.ErrUnbindingNotAllowed))
		return
	}

	err = dao.UpdateUserDeviceInfo(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func DeviceUpdateHandler(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	deviceInfo.UserID = c.Query("user_id")
	deviceInfo.DeviceName = c.Query("device_name")

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	if old != nil && old.UserID != "" {
		c.JSON(http.StatusBadRequest, respError(errors.ErrDeviceExists))
		return
	}

	err = dao.UpdateDeviceName(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusBadRequest, respError(errors.ErrInternalServer))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}
