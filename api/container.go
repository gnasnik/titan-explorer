package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	ctypes "github.com/Filecoin-Titan/titan-container/api/types"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/geo"
	"golang.org/x/xerrors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func GetProvidersHandler(c *gin.Context) {
	url := config.Cfg.ContainerManager.Addr
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("size"), 10, 64)

	params := ctypes.GetProviderOption{
		State: []ctypes.ProviderState{ctypes.ProviderStateOnline, ctypes.ProviderStateOffline, ctypes.ProviderStateAbnormal},
		Page:  int(page),
		Size:  int(size),
	}

	providers, err := getProvidersJsonRPC(url, params)
	if err != nil {
		log.Errorf("get providers: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	type result struct {
		ID      string `json:"id"`
		IP      string `json:"ip"`
		State   string `json:"state"`
		Host    string `json:"host"`
		CPU     string `json:"cpu"`
		Memory  string `json:"memory"`
		Storage string `json:"storage"`
		Region  string `json:"region"`
	}

	res := make([]result, 0)
	for _, provider := range providers {
		resource, err := getProviderStatisticJsonRPC(url, provider.ID)
		if err != nil {
			log.Errorf("get statistic %s: %v", provider.ID, err)
			continue
		}

		location, err := geo.GetIpLocation(c.Request.Context(), provider.IP)
		if err != nil {
			log.Errorf("get location: %v", err)
		}

		if location == nil {
			location = &model.Location{}
		}

		res = append(res, result{
			ID:      string(provider.ID),
			IP:      provider.IP,
			State:   ctypes.ProviderStateString(provider.State),
			Host:    provider.HostURI,
			CPU:     fmt.Sprintf("%.1f/%.1f", resource.CPUCores.Available, resource.CPUCores.MaxCPUCores),
			Memory:  fmt.Sprintf("%s/%s", units.BytesSize(float64(resource.Memory.Available)), units.BytesSize(float64(resource.Memory.MaxMemory))),
			Storage: fmt.Sprintf("%s/%s", units.BytesSize(float64(resource.Storage.Available)), units.BytesSize(float64(resource.Storage.MaxStorage))),
			Region:  fmt.Sprintf("%s %s", location.Country, location.City),
		})
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"providers": res,
	}))
}

func GetDeploymentsHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("size"), 10, 64)

	params := ctypes.GetDeploymentOption{
		Owner: username,
		Page:  int(page),
		Size:  int(size),
	}

	resp, err := getDeploymentsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get deployments: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	//type result struct {
	//	ID          string  `json:"id"`
	//	Name        string  `json:"name"`
	//	Image       string  `json:"image"`
	//	State       string  `json:"state"`
	//	Total       int     `json:"total"`
	//	Ready       int     `json:"ready"`
	//	Available   int     `json:"available"`
	//	CPU         float64 `json:"cpu"`
	//	GPU         float64 `json:"gpu"`
	//	Memory      string  `json:"memory"`
	//	Storage     string  `json:"storage"`
	//	Provider    string  `json:"provider"`
	//	Port        string  `json:"port"`
	//	CreatedTime string  `json:"created_time"`
	//}
	//
	//out := make([]ctypes.GetDeploymentListResp, 0)
	//
	//for _, deployment := range resp.Deployments {
	//	for _, service := range deployment.Services {
	//		state := ctypes.DeploymentStateInActive
	//		if service.Status.TotalReplicas == service.Status.ReadyReplicas {
	//			state = ctypes.DeploymentStateActive
	//		}
	//
	//		var exposePorts []string
	//		for _, port := range service.Ports {
	//			exposePorts = append(exposePorts, fmt.Sprintf("%d->%d", port.Port, port.ExposePort))
	//		}
	//
	//		var storageSize int64
	//		for _, storage := range service.Storage {
	//			storageSize += storage.Quantity
	//		}
	//
	//		out = append(out, result{
	//			ID:          string(deployment.ID),
	//			Name:        deployment.Name,
	//			Image:       service.Image,
	//			State:       ctypes.DeploymentStateString(state),
	//			Total:       service.Status.TotalReplicas,
	//			Ready:       service.Status.ReadyReplicas,
	//			Available:   service.Status.AvailableReplicas,
	//			CPU:         service.CPU,
	//			Memory:      units.BytesSize(float64(service.Memory * units.MiB)),
	//			Storage:     units.BytesSize(float64(storageSize * units.MiB)),
	//			Provider:    string(deployment.ProviderID),
	//			Port:        strings.Join(exposePorts, " "),
	//			CreatedTime: deployment.CreatedAt.Format(time.DateTime),
	//		})
	//	}
	//}

	c.JSON(http.StatusOK, respJSON(resp))
}

func GetDeploymentManifestHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")

	params := ctypes.GetDeploymentOption{
		Owner:        username,
		DeploymentID: ctypes.DeploymentID(deploymentId),
	}

	resp, err := getDeploymentsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get providers: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"deployment": resp.Deployments[0],
	}))
}

func CreateDeploymentHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	var deployment ctypes.Deployment
	err := c.BindJSON(&deployment)
	if err != nil {
		log.Errorf("%v", err)
		c.JSON(http.StatusBadRequest, respErrorCode(errors.InvalidParams, c))
		return
	}

	deployment.Owner = username
	url := config.Cfg.ContainerManager.Addr
	err = createDeploymentsJsonRPC(url, deployment)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusOK, respError(errors.InvalidParams, err))
			return
		}

		log.Errorf("create deployment: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func DeleteDeploymentHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	var deployment ctypes.Deployment
	if err := c.BindJSON(&deployment); err != nil {
		log.Errorf("%v", err)
		c.JSON(http.StatusBadRequest, respErrorCode(errors.InvalidParams, c))
		return
	}

	deployment.Owner = username
	url := config.Cfg.ContainerManager.Addr
	err := deleteDeploymentsJsonRPC(url, deployment)
	if err != nil {
		log.Errorf("delete deployment: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func UpdateDeploymentHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	var deployment ctypes.Deployment
	if err := c.BindJSON(&deployment); err != nil {
		log.Errorf("%v", err)
		c.JSON(http.StatusBadRequest, respErrorCode(errors.InvalidParams, c))
		return
	}

	deployment.Owner = username
	url := config.Cfg.ContainerManager.Addr
	err := updateDeploymentsJsonRPC(url, deployment)
	if err != nil {
		log.Errorf("update deployment: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func GetDeploymentLogsHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")

	params := ctypes.Deployment{
		Owner: username,
		ID:    ctypes.DeploymentID(deploymentId),
	}

	logs := make([]*ctypes.ServiceLog, 0)

	events, err := getDeploymentEventsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get events: %v", err)
		//c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		//return
	}

	for _, event := range events {
		l := &ctypes.ServiceLog{
			ServiceName: event.ServiceName,
		}
		for _, e := range event.Events {
			l.Logs = append(l.Logs, ctypes.Log(e))
		}
		logs = append(logs, l)
	}

	slogs, err := getDeploymentLogsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get logs: %v", err)
		//c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		//return
	}

	logs = append(logs, slogs...)

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"logs": logs,
	}))
}

func GetDeploymentEventsHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")

	params := ctypes.Deployment{
		Owner: username,
		ID:    ctypes.DeploymentID(deploymentId),
	}

	logs, err := getDeploymentEventsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get events: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"events": logs,
	}))
}

func GetDeploymentDomainHandler(c *gin.Context) {
	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")

	out := make([]*ctypes.DeploymentDomain, 0)
	domains, err := getDeploymentDomainJsonRPC(url, ctypes.DeploymentID(deploymentId))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Errorf("get domains: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	out = append(out, domains...)
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"domains": out,
	}))
}

func AddDeploymentDomainHandler(c *gin.Context) {
	url := config.Cfg.ContainerManager.Addr
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	type domainReq struct {
		ID   ctypes.DeploymentID
		Host string
		Key  string
		Cert string
	}

	var params domainReq
	if err := c.BindJSON(&params); err != nil {
		log.Errorf("%v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	dparam := ctypes.GetDeploymentOption{
		Owner:        username,
		DeploymentID: ctypes.DeploymentID(params.ID),
	}

	resp, err := getDeploymentsJsonRPC(url, dparam)
	if err != nil {
		log.Errorf("get providers: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if len(resp.Deployments) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	if resp.Deployments[0].Owner != username {
		c.JSON(http.StatusOK, respErrorCode(errors.PermissionNotAllowed, c))
		return
	}
	//
	//host := strings.Trim(params.Host, "https://")
	//host = strings.Trim(host, "http://")

	cert := &ctypes.Certificate{
		Host: params.Host,
		Key:  []byte(params.Key),
		Cert: []byte(params.Cert),
	}

	err = addDeploymentDomainJsonRPC(url, params.ID, cert)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Errorf("add domains: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func DeleteDeploymentDomainHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")
	host := c.Query("host")

	dparam := ctypes.GetDeploymentOption{
		Owner:        username,
		DeploymentID: ctypes.DeploymentID(deploymentId),
	}

	resp, err := getDeploymentsJsonRPC(url, dparam)
	if err != nil {
		log.Errorf("get providers: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if len(resp.Deployments) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	if resp.Deployments[0].Owner != username {
		c.JSON(http.StatusOK, respErrorCode(errors.PermissionNotAllowed, c))
		return
	}

	err = deleteDeploymentDomainJsonRPC(url, ctypes.DeploymentID(deploymentId), host)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Errorf("delete domains: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func GetDeploymentShellHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	url := config.Cfg.ContainerManager.Addr
	deploymentId := c.Query("id")

	params := ctypes.GetDeploymentOption{
		Owner:        username,
		DeploymentID: ctypes.DeploymentID(deploymentId),
	}

	resp, err := getDeploymentsJsonRPC(url, params)
	if err != nil {
		log.Errorf("get providers: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if len(resp.Deployments) == 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	if resp.Deployments[0].Owner != username {
		c.JSON(http.StatusOK, respErrorCode(errors.NotFound, c))
		return
	}

	shell, err := getDeploymentShellJsonRPC(url, ctypes.DeploymentID(deploymentId))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Errorf("get shell: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"shell": shell,
	}))
}

func getProvidersJsonRPC(url string, opt ctypes.GetProviderOption) ([]*ctypes.Provider, error) {
	params, err := json.Marshal([]interface{}{opt})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetProviderList",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var providers []*ctypes.Provider
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &providers)
	if err != nil {
		return nil, err
	}

	return providers, nil
}

func getProviderStatisticJsonRPC(url string, id ctypes.ProviderID) (*ctypes.ResourcesStatistics, error) {
	params, err := json.Marshal([]interface{}{id})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetStatistics",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var statistic ctypes.ResourcesStatistics
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &statistic)
	if err != nil {
		return nil, err
	}

	return &statistic, nil
}

func getDeploymentsJsonRPC(url string, opt ctypes.GetDeploymentOption) (*ctypes.GetDeploymentListResp, error) {
	params, err := json.Marshal([]interface{}{opt})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetDeploymentList",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var out ctypes.GetDeploymentListResp
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func createDeploymentsJsonRPC(url string, deployment ctypes.Deployment) error {
	params, err := json.Marshal([]interface{}{deployment})
	if err != nil {
		return err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.CreateDeployment",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return err
	}

	return nil
}

func deleteDeploymentsJsonRPC(url string, deployment ctypes.Deployment) error {
	params, err := json.Marshal([]interface{}{deployment, true})
	if err != nil {
		return err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.CloseDeployment",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return err
	}

	return nil
}

func updateDeploymentsJsonRPC(url string, deployment ctypes.Deployment) error {
	params, err := json.Marshal([]interface{}{deployment})
	if err != nil {
		return err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.UpdateDeployment",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return err
	}

	if rsp.Error != nil {
		return err
	}

	return nil
}

func getDeploymentLogsJsonRPC(url string, deployment ctypes.Deployment) ([]*ctypes.ServiceLog, error) {
	params, err := json.Marshal([]interface{}{deployment})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetLogs",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var logs []*ctypes.ServiceLog
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &logs)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func getDeploymentEventsJsonRPC(url string, deployment ctypes.Deployment) ([]*ctypes.ServiceEvent, error) {
	params, err := json.Marshal([]interface{}{deployment})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetEvents",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var event []*ctypes.ServiceEvent
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func getDeploymentDomainJsonRPC(url string, id ctypes.DeploymentID) ([]*ctypes.DeploymentDomain, error) {
	params, err := json.Marshal([]interface{}{id})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetDeploymentDomains",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var domains []*ctypes.DeploymentDomain
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &domains)
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func addDeploymentDomainJsonRPC(url string, id ctypes.DeploymentID, cert *ctypes.Certificate) error {
	params, err := json.Marshal([]interface{}{id, cert})
	if err != nil {
		return err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.AddDeploymentDomain",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return err
	}

	var domains []*ctypes.DeploymentDomain
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &domains)
	if err != nil {
		return err
	}

	return nil
}

func deleteDeploymentDomainJsonRPC(url string, id ctypes.DeploymentID, host string) error {
	params, err := json.Marshal([]interface{}{id, host})
	if err != nil {
		return err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.DeleteDeploymentDomain",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return err
	}

	var domains []*ctypes.DeploymentDomain
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &domains)
	if err != nil {
		return err
	}

	return nil
}

func getDeploymentShellJsonRPC(url string, id ctypes.DeploymentID) (*ctypes.ShellEndpoint, error) {
	params, err := json.Marshal([]interface{}{id})
	if err != nil {
		return nil, err
	}

	req := model.LotusRequest{
		Jsonrpc: "2.0",
		Method:  "titan.GetDeploymentShellEndpoint",
		Params:  params,
		ID:      1,
	}

	rsp, err := requestJsonRPC(url, req)
	if err != nil {
		return nil, err
	}

	var endpoint ctypes.ShellEndpoint
	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &endpoint)
	if err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func requestJsonRPC(url string, req model.LotusRequest) (*model.LotusResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	token := config.Cfg.ContainerManager.Token
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(request)
	//resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	//if err != nil {
	//	return nil, err
	//}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(body))

	var rsp model.LotusResponse
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, xerrors.New(rsp.Error.Message)
	}

	return &rsp, nil
}
