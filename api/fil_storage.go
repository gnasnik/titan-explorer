package api

import (
	"context"
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/filecoin"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/iptool"
	"github.com/multiformats/go-multiaddr"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func CreateFilStorageHandler(c *gin.Context) {
	var params []*model.FilStorage
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	for i := 0; i < len(params); i++ {
		params[i].StartTime = time.Unix(filecoin.GetTimestampByHeight(params[i].StartHeight), 0)
		params[i].EndTime = time.Unix(filecoin.GetTimestampByHeight(params[i].EndHeight), 0)
		params[i].CreatedAt = time.Now()
		params[i].UpdatedAt = time.Now()
		if params[i].FIndex == 0 {
			params[i].FIndex = int64(i)
		}

		sp, err := dao.GetStorageProvider(c.Request.Context(), params[i].Provider)
		if err == nil && sp != nil {
			params[i].IP = sp.IP
			continue
		}

		err = SaveProviderLocation(params[i].Provider)
		if err != nil {
			log.Errorf("SaveProviderLocation: %v", err)
		}
	}

	if err := dao.AddFilStorages(c.Request.Context(), params); err != nil {
		log.Errorf("add fil storage: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func GetFilStorageListHandler(c *gin.Context) {
	cid := c.Query("cid")
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	lang := c.GetHeader("lang")

	asset, err := dao.GetAssetByCID(c.Request.Context(), cid)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respJSON(JsonObject{
			"list":  nil,
			"total": 0,
		}))
		return
	}

	if err != nil {
		log.Errorf("GetAssetByCID: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	option := dao.QueryOption{
		Page:     page,
		PageSize: pageSize,
		Lang:     model.Language(lang),
	}

	list, total, err := dao.ListFilStorages(c.Request.Context(), asset.Path, option)
	if err != nil {
		log.Errorf("ListFilStorages: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  list,
		"total": total,
	}))
}

func SaveProviderLocation(providerId string) error {
	minerInfo, err := filecoin.StateMinerInfo(config.Cfg.FilecoinRPCServerAddress, providerId)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if len(minerInfo.MultiAddress) == 0 {
		return dao.AddStorageProvider(ctx, &model.StorageProvider{
			ProviderID:  providerId,
			IP:          "",
			Retrievable: false,
			Location:    "",
			CreatedAt:   time.Now(),
		})
	}

	mad, err := multiaddr.NewMultiaddrBytes(minerInfo.MultiAddress[0])
	if err != nil {
		return err
	}

	ip := strings.Split(mad.String(), "/")[2]

	var loc *model.Location
	language := []model.Language{model.LanguageEN, model.LanguageCN}
	for _, lang := range language {
		l, err := iptool.IPDataCloudGetLocation(ctx, config.Cfg.IpDataCloud.Url, ip, config.Cfg.IpDataCloud.Url, string(lang))
		if err != nil {
			return err
		}

		err = dao.UpsertLocationInfo(ctx, l, lang)
		if err != nil {
			return err
		}

		if loc == nil {
			loc = l
		}
	}

	return dao.AddStorageProvider(ctx, &model.StorageProvider{
		ProviderID:  providerId,
		IP:          ip,
		Retrievable: true,
		Location:    loc.CountryCode,
		CreatedAt:   time.Now(),
	})
}

func GetBackupAssetsHandler(c *gin.Context) {
	ctx := context.Background()
	assets, total, err := dao.GetAssetsByEmptyPath(ctx)
	if err != nil {
		log.Errorf("GetAssertsByEmptyPath: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  assets,
		"total": total,
	}))
}

func BackupResultHandler(c *gin.Context) {
	var params []*model.Asset
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	for _, assets := range params {
		if assets.Path != "" {
			err := dao.UpdateAssetPath(c.Request.Context(), assets.Cid, assets.Path)
			if err != nil {
				log.Errorf("update assets path: %v", err)
			}
			continue
		}

		err := dao.UpdateAssetBackupResult(c.Request.Context(), assets.Cid, int(assets.BackupResult))
		if err != nil {
			log.Errorf("update assets backup result: %v", err)
		}
	}

	c.JSON(http.StatusOK, respJSON(nil))
}
