package statistics

import (
	"context"
	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/gnasnik/titan-explorer/core/geo"
	"github.com/gnasnik/titan-explorer/pkg/formatter"
	"github.com/oschwald/geoip2-golang"
	"net"
	"strconv"
	"time"

	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
)

const maxPageSize = 500

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
		err := dao.BulkUpsertDeviceInfo(ctx, onlineNodes)
		if err != nil {
			log.Errorf("bulk upsert device info: %v", err)
		}

		if err = addDeviceInfoHours(ctx, onlineNodes); err != nil {
			log.Errorf("add device info hours: %v", err)
		}

		err = dao.BulkAddDeviceInfo(ctx, offlineNodes)
		if err != nil {
			log.Errorf("bulk add device info: %v", err)
		}
		return nil
	})

	if total < resp.Total {
		goto loop
	}

	// add inactive node records for statistics
	//e := dao.GenerateInactiveNodeRecords(context.Background(), start)
	//if e != nil {
	//	log.Errorf("generate inactive node records: %v", e)
	//}

	return nil
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
	case 0:
		deviceInfo.DeviceStatus = DeviceStatusOffline
		deviceInfo.DeviceID = node.NodeID
		deviceInfo.DeviceStatusCode = DeviceStatusCodeOffline
	case 1, 2:
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
