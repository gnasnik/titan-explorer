package statistics

import (
	"context"
	"fmt"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gnasnik/titan-explorer/core/geo"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/golang-module/carbon/v2"
	"github.com/oschwald/geoip2-golang"
	errs "github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"net"
	"strconv"
	"time"

	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
)

const maxPageSize = 100

const (
	DeviceStatusOffline  = "offline"
	DeviceStatusOnline   = "online"
	DeviceStatusAbnormal = "abnormal"

	DeviceStatusCodeOffline  = 3
	DeviceStatusCodeOnline   = 1
	DeviceStatusCodeAbnormal = 2
)

// NodeFetcher handles fetching information about all nodes
type NodeFetcher struct {
	BaseFetcher
}

func init() {
	// Register newNodeFetcher during initialization
	RegisterFetcher(newNodeFetcher)
}

// newNodeFetcher creates a new NodeFetcher instance
func newNodeFetcher() Fetcher {
	return &NodeFetcher{BaseFetcher: newBaseFetcher()}
}

// Fetch fetches information about all nodes
func (n *NodeFetcher) Fetch(ctx context.Context, scheduler *Scheduler) error {
	log.Info("start fetching all nodes")
	start := time.Now()
	defer func() {
		log.Infof("fetch all nodes done, cost: %v", time.Since(start))
	}()

	var total int64
	page, size := 1, maxPageSize

loop:
	offset := (page - 1) * size
	resp, err := scheduler.Api.GetNodeList(ctx, offset, size)
	if err != nil {
		log.Errorf("api ListNodes: %v", err)
		return nil
	}

	total += int64(len(resp.Data))
	page++

	var (
		onlineNodes  []*model.DeviceInfo
		offlineNodes []*model.DeviceInfo
	)

	for _, node := range resp.Data {
		if node.NodeID == "" {
			continue
		}

		nodeInfo := ToDeviceInfo(ctx, node)
		if nodeInfo.DeviceStatus == DeviceStatusOffline {
			// just update device status
			err = dao.UpdateDeviceStatus(ctx, nodeInfo)
			if err != nil {
				log.Errorf("update device status: %v", err)
			}

			offlineNodes = append(offlineNodes, nodeInfo)
			continue
		}

		onlineNodes = append(onlineNodes, nodeInfo)
	}

	if len(onlineNodes)+len(offlineNodes) < 1 {
		log.Errorf("start to fetch all nodes: nodes length is 0")
		return nil
	}

	log.Infof("handling %d/%d nodes, online: %d offline: %d", total, resp.Total, len(onlineNodes), len(offlineNodes))

	n.Push(ctx, func() error {
		if len(onlineNodes) > 0 {
			err := dao.BulkUpsertDeviceInfo(ctx, onlineNodes)
			if err != nil {
				log.Errorf("bulk upsert device info: %v", err)
			}

			if err = addDeviceInfoHours(ctx, onlineNodes); err != nil {
				log.Errorf("add device info hours: %v", err)
			}

		}

		if len(offlineNodes) > 0 {
			err = dao.BulkAddDeviceInfo(ctx, offlineNodes)
			if err != nil {
				log.Errorf("bulk add device info: %v", err)
			}
		}

		return nil
	})

	if total < resp.Total {
		goto loop
	}

	// finally summary the data
	n.Push(ctx, func() error {
		var eg errgroup.Group

		eg.Go(SumDeviceInfoProfit)
		eg.Go(SumAllNodes)
		eg.Go(UpdateDeviceRank)

		if err := eg.Wait(); err != nil {
			log.Errorf("sumary job: %v", err)
		}

		return err
	})

	// add inactive node records for statistics
	//e := dao.GenerateInactiveNodeRecords(context.Background(), start)
	//if e != nil {
	//	log.Errorf("generate inactive node records: %v", e)
	//}

	return nil
}

func sumDailyReward(ctx context.Context, sumTime time.Time, devices []*model.DeviceInfo) error {
	// query before today value
	start := carbon.Parse("2024-03-01").String()
	end := carbon.Yesterday().EndOfDay().String()

	var deviceIds []string
	for _, device := range devices {
		deviceIds = append(deviceIds, device.DeviceID)
	}

	// query before today device reward
	maxDeviceInfos, err := dao.QueryMaxDeviceDailyInfo(ctx, deviceIds, start, end)
	if err != nil {
		log.Errorf("QueryMaxDeviceDailyInfo: %v", err)
		return err
	}

	var updatedDevices []*model.DeviceInfoDaily
	for _, deviceInfo := range devices {
		deviceInfo.UpdatedAt = sumTime
		ud, ok := maxDeviceInfos[deviceInfo.DeviceID]
		if !ok {
			updatedDevices = append(updatedDevices, deviceInfoToDailyInfo(deviceInfo))
			continue
		}

		deviceInfo.CumulativeProfit = deviceInfo.CumulativeProfit - ud.Income
		deviceInfo.OnlineTime = deviceInfo.OnlineTime - ud.OnlineTime
		deviceInfo.UploadTraffic = deviceInfo.UploadTraffic - ud.UpstreamTraffic
		deviceInfo.DownloadTraffic = deviceInfo.DownloadTraffic - ud.DownstreamTraffic
		deviceInfo.RetrievalCount = deviceInfo.RetrievalCount - ud.RetrievalCount

		updatedDevices = append(updatedDevices, deviceInfoToDailyInfo(deviceInfo))
	}

	// update cumulative profit

	err = dao.BulkUpsertDeviceInfoDaily(context.Background(), updatedDevices)
	if err != nil {
		return errs.Wrap(err, "bulk upsert device info daily")
	}

	return nil
}

func deviceInfoToDailyInfo(deviceInfo *model.DeviceInfo) *model.DeviceInfoDaily {

	return &model.DeviceInfoDaily{
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		UserID:            deviceInfo.UserID,
		DeviceID:          deviceInfo.DeviceID,
		Time:              carbon.Time2Carbon(deviceInfo.UpdatedAt).StartOfDay().AddHours(8).Carbon2Time(),
		Income:            deviceInfo.CumulativeProfit,
		OnlineTime:        deviceInfo.OnlineTime,
		PkgLossRatio:      0, // todo
		Latency:           0, //todo
		NatRatio:          0,
		DiskUsage:         deviceInfo.DiskUsage,
		DiskSpace:         deviceInfo.DiskSpace,
		BandwidthUp:       deviceInfo.BandwidthUp,
		BandwidthDown:     deviceInfo.BandwidthDown,
		UpstreamTraffic:   deviceInfo.UploadTraffic,
		DownstreamTraffic: deviceInfo.DownloadTraffic,
		RetrievalCount:    deviceInfo.RetrievalCount,
		BlockCount:        0,
	}
}

func calculateDailyInfo(ctx context.Context, start, end string) ([]map[string]string, error) {
	where := fmt.Sprintf("where 1=1", start, end)
	if start != "" {
		where += fmt.Sprintf(" and time>='%s'", start)
	}
	if end != "" {
		where += fmt.Sprintf(" and time <'%s'", end)
	}

	//where := fmt.Sprintf("where time>='%s' and time<='%s'", start, end)
	var total int64
	err := dao.DB.GetContext(ctx, &total, fmt.Sprintf(`SELECT count(*) FROM %s %s`, "device_info_hour", where))
	if err != nil {
		return nil, err
	}

	sqlClause := fmt.Sprintf(`select i.user_id, i.device_id, date_format(i.time, '%%Y-%%m-%%d') as date,
			i.nat_ratio, i.disk_usage, i.disk_space,i.latency, i.pkg_loss_ratio, i.bandwidth_up, i.bandwidth_down,
			max(i.hour_income) as hour_income,
			max(i.online_time) as online_time,
			max(i.upstream_traffic) as upstream_traffic,
			max(i.downstream_traffic) as downstream_traffic,
			max(i.retrieval_count) as retrieval_count,
			max(i.block_count) as block_count
			from (select * from device_info_hour %s order by id desc limit %d) i
			group by device_id`, where, total)

	dataList, err := dao.GetQueryDataList(sqlClause)
	if err != nil {
		return nil, errs.Wrap(err, "get query data list")
	}

	return dataList, nil
}

func ToDeviceInfo(ctx context.Context, node types.NodeInfo) *model.DeviceInfo {
	deviceInfo := model.DeviceInfo{
		DeviceID:         node.NodeID,
		DeviceName:       node.NodeName,
		DiskSpace:        formatter.ToFixed(node.DiskSpace, 2),
		DiskUsage:        formatter.ToFixed(node.DiskUsage, 2),
		ActiveStatus:     1,
		OnlineTime:       float64(node.OnlineDuration),
		BandwidthUp:      float64(node.BandwidthUp),
		BandwidthDown:    float64(node.BandwidthDown),
		CpuUsage:         formatter.ToFixed(node.CPUUsage, 2),
		CpuCores:         int64(node.CPUCores),
		Memory:           node.Memory,
		MemoryUsage:      formatter.ToFixed(node.MemoryUsage, 2),
		UploadTraffic:    float64(node.UploadTraffic),
		DownloadTraffic:  float64(node.DownloadTraffic),
		ExternalIp:       node.ExternalIP,
		InternalIp:       node.InternalIP,
		IoSystem:         node.IoSystem,
		SystemVersion:    node.SystemVersion,
		DiskType:         node.DiskType,
		CumulativeProfit: node.Profit,
		NodeType:         int64(node.Type),
		CacheCount:       node.AssetCount,
		RetrievalCount:   node.RetrieveCount,
		NATType:          node.NATType,
		UpdatedAt:        node.LastSeen,
		BoundAt:          node.FirstTime,
		IncomeIncr:       node.IncomeIncr,
	}

	switch node.Status {
	case types.NodeOffline:
		deviceInfo.DeviceStatus = DeviceStatusOffline
		deviceInfo.DeviceID = node.NodeID
		deviceInfo.DeviceStatusCode = DeviceStatusCodeOffline
	case types.NodeServicing, types.NodeNatSymmetric:
		deviceInfo.DeviceStatusCode = DeviceStatusCodeOnline
		deviceInfo.DeviceStatus = DeviceStatusOnline
	default:
		deviceInfo.DeviceStatusCode = DeviceStatusCodeAbnormal
		deviceInfo.DeviceStatus = DeviceStatusAbnormal
	}

	applyLocationInfo(&deviceInfo)

	return &deviceInfo
}

func applyLocationInfo(deviceInfo *model.DeviceInfo) {
	if deviceInfo.ExternalIp == "" {
		return
	}

	//var loc model.Location
	//err := GetIpLocation(context.Background(), deviceInfo.ExternalIp, &loc, model.LanguageCN, model.LanguageEN)
	loc, err := geo.GetIpLocation(context.Background(), deviceInfo.ExternalIp, model.LanguageEN)
	if err != nil || loc == nil {
		log.Errorf("%v", err)
		applyLocationFromLocalGEODB(deviceInfo)
		return
	}

	deviceInfo.NetworkInfo = loc.Isp
	deviceInfo.IpProvince = loc.Province
	deviceInfo.IpCountry = loc.Country
	deviceInfo.IpCity = loc.City
	deviceInfo.IpLocation = dao.ContactIPLocation(*loc, model.LanguageEN)
	deviceInfo.Longitude, _ = strconv.ParseFloat(loc.Longitude, 64)
	deviceInfo.Latitude, _ = strconv.ParseFloat(loc.Latitude, 64)
}

func applyLocationFromLocalGEODB(deviceInfo *model.DeviceInfo) {
	db, err := geoip2.Open("city.mmdb")
	if err != nil {
		log.Errorf("open city.mmdb: %v", err)
		return
	}
	defer db.Close()

	// If you are using strings that may be invalid, check that ip is not nil
	if deviceInfo.ExternalIp == "" {
		return
	}

	ip := net.ParseIP(deviceInfo.ExternalIp)
	record, err := db.City(ip)
	if err != nil {
		log.Errorf("query ip %s: %v", deviceInfo.ExternalIp, err)
		return
	}

	if len(record.Subdivisions) > 0 {
		deviceInfo.IpProvince = record.Subdivisions[0].Names["en"]
	}

	continent := record.Continent.Names["en"]
	deviceInfo.IpCountry = record.Country.Names["en"]
	deviceInfo.IpCity = record.City.Names["en"]
	deviceInfo.IpLocation = continent + "-" + deviceInfo.IpCountry + "-" + deviceInfo.IpProvince
	if deviceInfo.IpCity != "" {
		deviceInfo.IpLocation += "-" + deviceInfo.IpCity
	}

	deviceInfo.Longitude = record.Location.Longitude
	deviceInfo.Latitude = record.Location.Latitude

	return
}

var _ Fetcher = &NodeFetcher{}
