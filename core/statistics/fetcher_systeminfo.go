package statistics

import (
	"context"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"time"
)

type SystemInfoFetcher struct {
	BaseFetcher
}

func newSystemInfoFetcher() *SystemInfoFetcher {
	return &SystemInfoFetcher{BaseFetcher: newBaseFetcher()}
}

func (s *SystemInfoFetcher) Fetch(ctx context.Context, scheduler *Scheduler) error {
	log.Info("start to fetch system info")
	start := time.Now()
	defer func() {
		log.Infof("count fetch system info done, cost: %v", time.Since(start))
	}()

	resp, err := scheduler.Api.GetSystemInfo(ctx)
	if err != nil {
		log.Errorf("api GetSystemInfo: %v", err)
		return err
	}

	s.Push(ctx, func() error {
		if err := dao.UpsertSystemInfo(ctx, &model.SystemInfo{
			SchedulerUuid:    scheduler.Uuid,
			CarFileCount:     int64(resp.CarFileCount),
			DownloadCount:    int64(resp.DownloadCount),
			NextElectionTime: resp.NextElectionTime,
		}); err != nil {
			log.Errorf("upsert system info: %v", err)
		}
		return nil
	})

	return nil
}

var _ Fetcher = &SystemInfoFetcher{}
