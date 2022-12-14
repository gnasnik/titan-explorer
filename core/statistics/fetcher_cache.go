package statistics

import (
	"context"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/utils"
	"github.com/golang-module/carbon/v2"
	"github.com/ipfs/go-cid"
	"github.com/linguohua/titan/api"
	mh "github.com/multiformats/go-multihash"
	"math"
	"time"
)

type CacheFetcher struct {
	BaseFetcher
}

func newCacheFetcher() *CacheFetcher {
	return &CacheFetcher{BaseFetcher: newBaseFetcher()}
}

func (c *CacheFetcher) Fetch(ctx context.Context, scheduler *Scheduler) error {
	log.Info("start to fetch cache files")
	start := time.Now()
	defer func() {
		log.Infof("fetch cache files, cost: %v", time.Since(start))
	}()

	var (
		startTime, endTime time.Time
		sum                int64
	)

	lastEvent, err := dao.GetLastCacheEvent(ctx)
	if err != nil {
		log.Errorf("get last cache event: %v", err)
		return err
	}

	if lastEvent == nil {
		startTime = carbon.Now().StartOfDay().StartOfMinute().Carbon2Time()
	} else {
		startTime = lastEvent.MaxCreatedTime.Add(time.Second)
	}

	endTime = carbon.Time2Carbon(start).SubMinutes(start.Minute() % 5).StartOfMinute().Carbon2Time()
	req := api.ListCacheBlocksReq{
		StartTime: startTime.Unix(),
		EndTime:   endTime.Unix(),
		Cursor:    0,
		Count:     maxPageSize,
	}

loop:
	resp, err := scheduler.Api.GetCacheBlockInfos(ctx, req)
	if err != nil {
		log.Errorf("client GetCacheBlockInfos: %v", err)
		return err
	}

	var blockInfos []*model.BlockInfo
	for _, blockInfo := range resp.Data {
		blockInfos = append(blockInfos, toBlockInfo(blockInfo))
	}

	if resp.Total <= 0 {
		return nil
	}

	sum += int64(len(resp.Data))
	req.Cursor += len(resp.Data)

	log.Debugf("GetCacheBlockInfos got %d/%d blocks", sum, resp.Total)

	c.Push(ctx, func() error {
		err = dao.CreateBlockInfo(ctx, blockInfos)
		if err != nil {
			log.Errorf("create block info: %v", err)
		}
		return nil
	})

	if sum < resp.Total {
		goto loop
	}

	c.Push(ctx, func() error {
		err = dao.TxStatisticDeviceBlocks(ctx, startTime, endTime)
		if err != nil {
			log.Errorf("statistics device blocks: %v", err)
			return err
		}
		return nil
	})

	return nil
}

var _ Fetcher = &CacheFetcher{}

func toBlockInfo(in api.BlockInfo) *model.BlockInfo {
	return &model.BlockInfo{
		DeviceID:    in.DeviceID,
		CarfileHash: in.CarfileHash,
		CarfileCid:  hashToCID(in.CarfileHash),
		Status:      int32(in.Status),
		Size:        utils.ToFixed(float64(in.Size)/megaBytes, 2),
		CreatedTime: in.CreateTime,
		EndTime:     in.EndTime,
	}
}

func toValidationEvent(in api.ValidateResultInfo) *model.ValidationEvent {
	return &model.ValidationEvent{
		DeviceID:        in.DeviceID,
		ValidatorID:     in.ValidatorID,
		Status:          int32(in.Status),
		Blocks:          in.BlockNumber,
		Time:            in.ValidateTime,
		Duration:        in.Duration,
		UpstreamTraffic: utils.ToFixed(in.UploadTraffic/megaBytes, 2),
	}
}

func hashToCID(hashString string) string {
	multiphase, err := mh.FromHexString(hashString)
	if err != nil {
		return ""
	}
	cid := cid.NewCidV1(cid.Raw, multiphase)
	return cid.String()
}

func (s *Statistic) CountRetrievals() error {
	log.Info("start to count retrievals")
	start := time.Now()
	defer func() {
		log.Infof("count retrievals, cost: %v", time.Since(start))
	}()

	var startTime time.Time
	ctx := context.Background()
	lastEvent, err := dao.GetLastRetrievalEvent(ctx)
	if err != nil {
		log.Errorf("get last retrieval event: %v", err)
		return err
	}

	if lastEvent == nil {
		startTime = carbon.Now().SubDays(60).Carbon2Time()
	} else {
		startTime = floorFiveMinute(lastEvent.Time)
	}

	now := time.Now()
	oneDay := 24 * time.Hour
	for st := startTime; st.Before(now); {
		startT := st
		st = st.Add(oneDay)
		endT := st
		events, err := dao.GenerateRetrievalEvents(ctx, startT, endT)
		if err != nil {
			log.Errorf("generate retrieval events: %v", err)
			continue
		}

		if len(events) == 0 {
			continue
		}

		err = dao.CreateRetrievalEvents(ctx, events)
		if err != nil {
			log.Errorf("create retrieve events %v", err)
			continue
		}
	}

	return nil
}

func floorFiveMinute(t time.Time) time.Time {
	year, month, day := t.Date()
	hour := t.Hour()
	minute := int(5 * (math.Floor(float64(t.Minute() / 5))))
	return time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}
