package statistics

import (
	"context"
	"database/sql"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/golang-module/carbon/v2"
	errs "github.com/pkg/errors"
	"time"
)

const (
	defaultRequestLimit = 500
	defaultBackupDays   = 7
)

// AssertFetcher represents a fetcher for asset information.
type AssertFetcher struct {
	BaseFetcher
}

// Register the AssertFetcher during initialization
func init() {
	RegisterFetcher(newAssertFetcher)
}

// newAssertFetcher creates a new instance of AssertFetcher.
func newAssertFetcher() Fetcher {
	return &AssertFetcher{BaseFetcher: newBaseFetcher()}
}

// Fetch fetches asset information.
func (a AssertFetcher) Fetch(ctx context.Context, scheduler *Scheduler) error {
	log.Info("Start to fetch assets")
	start := time.Now()
	defer func() {
		log.Infof("fetch assets cost: %v", time.Since(start))
	}()

	latest, err := dao.GetLatestAsset(ctx)
	if err != nil && !errs.Is(err, sql.ErrNoRows) {
		return err
	}

	var (
		startTime, endTime time.Time
		limit, offset      int
	)

	if latest == nil || latest.EndTime.IsZero() {
		startTime = carbon.Now().StartOfDay().SubDays(defaultBackupDays).StdTime()
	} else {
		startTime = latest.EndTime
	}

	limit = defaultRequestLimit
	endTime = carbon.Now().EndOfDay().StdTime()

Loop:
	assetRecords, err := scheduler.Api.GetAssetRecordsByDateRange(ctx, offset, limit, startTime, endTime)
	if err != nil {
		log.Errorf("client api GetAssetRecordsByDateRange: %v", err)
		return err
	}

	offset += len(assetRecords.List)

	if len(assetRecords.List) == 1 && assetRecords.List[0].CID == latest.Cid {
		return nil
	}

	assets, err := toAssets(assetRecords.List, scheduler.AreaId)
	if err != nil {
		log.Errorf("toAssets: %v", err)
		return err
	}

	if len(assets) == 0 {
		return nil
	}

	err = dao.AddAssets(ctx, assets)
	if err != nil {
		log.Errorf("create user assets: %v", err)
	}

	if assetRecords.Total > int64(offset) {
		goto Loop
	}

	return nil
}

// toAssets converts a slice of ReplicaEventInfo to a slice of Asset.
func toAssets(in []*types.AssetRecord, areaId string) ([]*model.Asset, error) {
	var out []*model.Asset
	for _, r := range in {

		out = append(out, &model.Asset{
			Cid:                   r.CID,
			Hash:                  r.Hash,
			TotalSize:             r.TotalSize,
			TotalBlocks:           r.TotalBlocks,
			Expiration:            r.Expiration,
			CreatedTime:           r.CreatedTime,
			EndTime:               r.EndTime,
			UserId:                r.Owner,
			NeedCandidateReplicas: r.NeedCandidateReplicas,
			NeedEdgeReplica:       r.NeedEdgeReplica,
			NeedBandwidth:         r.NeedBandwidth,
			AreaId:                areaId,
			State:                 r.State,
			Note:                  r.Note,
			Source:                r.Source,
			RetryCount:            r.RetryCount,
			ReplenishReplicas:     r.ReplenishReplicas,
			FailedCount:           int64(r.FailedCount),
			SucceededCount:        int64(r.SucceededCount),
		})
	}
	return out, nil
}

func (a AssertFetcher) Finalize() error {
	return nil
}

var _ Fetcher = &AssertFetcher{}
