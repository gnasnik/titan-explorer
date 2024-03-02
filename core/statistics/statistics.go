package statistics

import (
	"fmt"
	"github.com/bsm/redislock"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	logging "github.com/ipfs/go-log/v2"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/context"
	"sync"
	"time"
)

var log = logging.Logger("statistics")

const LockerTTL = 30 * time.Second
const statisticLockerKeyPrefix = "TITAN::STATISTIC"

// FetcherRegistry to keep track of registered fetchers
var FetcherRegistry []func() Fetcher

// RegisterFetcher allows registering new fetchers
func RegisterFetcher(fetcher func() Fetcher) {
	FetcherRegistry = append(FetcherRegistry, fetcher)
}

// Statistic represents the statistics manager.
type Statistic struct {
	ctx        context.Context
	cfg        config.StatisticsConfig
	cron       *cron.Cron
	locker     *redislock.Client
	fetchers   []Fetcher
	schedulers []*Scheduler
}

// New creates a new Statistic instance.
func New(cfg config.StatisticsConfig, scheduler []*Scheduler) *Statistic {
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
	)

	s := &Statistic{
		ctx:        context.Background(),
		cron:       c,
		cfg:        cfg,
		schedulers: scheduler,
		locker:     redislock.New(dao.RedisCache),
		fetchers:   make([]Fetcher, 0),
	}

	for _, fetcher := range FetcherRegistry {
		s.fetchers = append(s.fetchers, fetcher())
	}

	return s
}

// Run starts the cron jobs for statistics.
func (s *Statistic) Run() {
	if s.cfg.Disable {
		return
	}
	s.cron.AddFunc(s.cfg.Crontab, s.Once("FETCHER", s.runFetchers))
	s.cron.Start()
	s.handleJobs()
}

func (s *Statistic) handleJobs() {
	for _, fetcher := range s.fetchers {
		go func(f Fetcher) {
			for {
				select {
				case job := <-f.GetJobQueue():
					if err := job(); err != nil {
						log.Errorf("run job: %v", err)
					}
				case <-s.ctx.Done():
					return
				}
			}
		}(fetcher)
	}
}

func (s *Statistic) runFetchers() error {
	var wg sync.WaitGroup
	wg.Add(len(s.schedulers))
	for _, scheduler := range s.schedulers {
		go func(scheduler *Scheduler) {
			defer wg.Done()
			for _, fetcher := range s.fetchers {
				err := fetcher.Fetch(s.ctx, scheduler)
				if err != nil {
					log.Errorf("run fetcher: %v", err)
				}
			}
		}(scheduler)
	}
	wg.Wait()

	s.asyncExecute([]func() error{
		s.SumDeviceInfoProfit,
		s.SumAllNodes,
		s.UpdateDeviceRank,
		//s.ClaimUserEarning,
	})

	return nil
}

// Stop stops the cron jobs and closes schedulers.
func (s *Statistic) Stop() {
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
	}
	for _, scheduler := range s.schedulers {
		scheduler.Closer()
	}
}

func (s *Statistic) Once(key string, fn func() error) func() {
	return func() {
		dKey := fmt.Sprintf("%s::%s", statisticLockerKeyPrefix, key)
		lock, err := s.locker.Obtain(s.ctx, dKey, LockerTTL, nil)
		if err == redislock.ErrNotObtained {
			log.Debugf("%s: %v", dKey, redislock.ErrNotObtained)
			return
		}

		if err != nil {
			log.Errorf("obtain redis lock: %v", err)
			return
		}

		defer lock.Release(s.ctx)
		if err = fn(); err != nil {
			log.Errorf("execute cron job: %v", err)
		}
	}
}

func (s *Statistic) asyncExecute(jobs []func() error) {
	var wg sync.WaitGroup
	wg.Add(len(jobs))
	for _, job := range jobs {
		go func(job func() error) {
			defer wg.Done()
			if err := job(); err != nil {
				log.Errorf("handling job: %v", err)
			}
		}(job)
	}
	wg.Wait()
}
