package statistics

import (
	"context"
	"database/sql"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/golang-module/carbon/v2"
	errs "github.com/pkg/errors"
	"sort"
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
	log.Info("Start to fetch assert info")
	start := time.Now()
	defer func() {
		log.Infof("Fetch assert files, cost: %v", time.Since(start))
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
		startTime = carbon.Now().StartOfDay().SubDays(defaultBackupDays).Carbon2Time()
	} else {
		startTime = latest.EndTime
	}

	limit = defaultRequestLimit
	endTime = carbon.Now().EndOfDay().Carbon2Time()

Loop:
	assertsRes, err := scheduler.Api.GetReplicaEvents(ctx, startTime, endTime, limit, offset)
	if err != nil {
		log.Errorf("client api GetReplicaEvents: %v", err)
		return err
	}

	offset += len(assertsRes.ReplicaEvents)

	if len(assertsRes.ReplicaEvents) == 1 && assertsRes.ReplicaEvents[0].Cid == latest.Cid {
		return nil
	}

	assets, err := toAssets(assertsRes.ReplicaEvents)
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

	if assertsRes.Total > offset {
		goto Loop
	}

	stats, err := dao.CountAssets(ctx)
	if err != nil {
		log.Errorf("count assets err: %v", err)
		return err
	}

	if len(stats) == 0 {
		return nil
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalSize > stats[j].TotalSize
	})

	for index, current := range stats {
		stats[index].Rank = int64(index) + 1

		storages, _, err := dao.ListStorageStats(ctx, current.ProjectId, dao.QueryOption{
			Page:      1,
			PageSize:  1,
			StartTime: carbon.Now().SubHours(24).String(),
			EndTime:   carbon.Now().String(),
		})
		if err != nil {
			log.Errorf("ListStorageStats: %v", err)
			continue
		}

		if len(storages) == 0 {
			continue
		}

		current.StorageChange24H = current.TotalSize - storages[0].TotalSize
		current.StorageChangePercentage24H = float64(current.TotalSize-storages[0].TotalSize) / float64(storages[0].TotalSize)
	}

	return dao.AddStorageStats(ctx, stats)
}

// toAssets converts a slice of ReplicaEventInfo to a slice of Asset.
func toAssets(in []*types.ReplicaEventInfo) ([]*model.Asset, error) {
	var out []*model.Asset
	for _, r := range in {
		out = append(out, &model.Asset{
			NodeID:     r.NodeID,
			Event:      int64(r.Event),
			Cid:        r.Cid,
			Hash:       r.Hash,
			TotalSize:  r.TotalSize,
			Expiration: r.Expiration,
			EndTime:    r.EndTime,
		})
	}
	return out, nil
}

var _ Fetcher = &AssertFetcher{}
