package powermax

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/kckecheng/powermax_exporter/common"
)

const (
	httpNoContent      = 204
	httpPartialContent = 206
)

// PowerMax array object
type PowerMax struct {
	APIEndPoint string
	User        string
	Password    string
	SymmID      string
	client      *resty.Client // Not exposed
}

type basePayload struct {
	SymmetrixID string   `json:"symmetrixId"`
	DataFormat  string   `json:"dataFormat"`
	Metrics     []string `json:"metrics"`
	StartDate   int64    `json:"startDate"`
	EndDate     int64    `json:"endDate"`
}

type baseResult struct {
	ResultList struct {
		Result []interface{} `json:"result"`
		From   int64         `json:"from"`
		To     int64         `json:"to"`
	} `json:"resultList"`
	ID             string `json:"id"`
	Count          int64  `json:"count"`
	ExpirationTime int64  `json:"expirationTime"`
	MaxPageSize    int64  `json:"maxPageSize"`
}

// NewPowerMax init a PowerMax object
func NewPowerMax(address, user, password, symmid string) *PowerMax {
	common.Logger.Infof("Init HTTP client to Unisphere %s", address)

	pm := PowerMax{}
	pm.APIEndPoint = fmt.Sprintf("https://%s:8443/univmax/restapi", address)
	pm.User = user
	pm.Password = password
	pm.SymmID = symmid
	pm.client = newClient(pm.User, pm.Password)

	arrays := pm.ListArrays()
	if arrays == nil || len(arrays) == 0 {
		common.Logger.Fatalf("No PowerMax is managed by Unisphere %s", address)
	}
	for _, a := range arrays {
		if symmid == a {
			return &pm
		}
	}
	common.Logger.Fatalf("PowerMax %s is not managed by Unisphere %s", symmid, address)
	return nil
}

// ListArrays list PowerMax managed by the Unisphere
func (pm PowerMax) ListArrays() []string {
	common.Logger.Info("List PowerMax managed by this Unisphere")

	resp, err := pm.Get("/performance/Array/keys", nil)
	if err != nil || resp.IsError() {
		reportIssues(resp, err)
		common.Logger.Fatal("Fail to list available arrays registered under the Unisphere")
	}

	var data = struct {
		ArrayInfo []struct {
			SymmetrixID        string `json:"symmetrixId"`
			FirstAvailableDate int64  `json:"firstAvailableDate"`
			LastAvailableDate  int64  `json:"lastAvailableDate"`
		} `json:"arrayInfo"`
	}{}
	err = json.Unmarshal(resp.Body(), &data)
	if err != nil {
		common.Logger.Fatal("Fail to decode json to get symmetrixid")
	}

	var ret []string
	for _, a := range data.ArrayInfo {
		ret = append(ret, a.SymmetrixID)
	}
	common.Logger.Debugf("PowerMax %+v are found under Unisphere", ret)
	return ret
}

// GetLatestArrayMetrics get the latest array overall metrics
func (pm PowerMax) GetLatestArrayMetrics(startTime, endTime int64) map[string]float64 {
	common.Logger.Infof("Get array metrics for %s from %d to %d", pm.SymmID, startTime, endTime)

	uri := "/performance/Array/metrics"
	payload := basePayload{
		SymmetrixID: pm.SymmID,
		DataFormat:  "Average",
		Metrics:     Metrics["array"],
		StartDate:   startTime,
		EndDate:     endTime,
	}
	resp, err := pm.Post(uri, payload)
	reportIssues(resp, err)

	return extractLatestMetrics(resp)
}

// ListCachePartitionIDs list cache partitions
func (pm PowerMax) ListCachePartitionIDs() []string {
	common.Logger.Infof("List cache partition for %s", pm.SymmID)

	uri := "/performance/CachePartition/keys"
	payload := map[string]string{"symmetrixId": pm.SymmID}
	return pm.listResourceIDs(uri, payload)
}

// GetLatestCacheMetrics get the latest cache partition metrics
func (pm PowerMax) GetLatestCacheMetrics(startTime, endTime int64, id string) map[string]float64 {
	common.Logger.Infof("Get latest cache parition metrics for %s:%s from %d to %d", pm.SymmID, id, startTime, endTime)

	uri := "/performance/CachePartition/metrics"
	payload := struct {
		basePayload
		CachePartitionID string `json:"cachePartitionId"`
	}{
		basePayload: basePayload{
			SymmetrixID: pm.SymmID,
			DataFormat:  "Average",
			Metrics:     Metrics["cache"],
			StartDate:   startTime,
			EndDate:     endTime,
		},
		CachePartitionID: id,
	}
	resp, err := pm.Post(uri, payload)
	reportIssues(resp, err)

	return extractLatestMetrics(resp)
}

// ListDirectorIDs list directors
func (pm PowerMax) ListDirectorIDs(dirType string) []string {
	common.Logger.Infof("List director IDs for %s:%s", pm.SymmID, dirType)

	urim := map[string]string{
		"fe": "/performance/FEDirector/keys",
		"be": "/performance/BEDirector/keys",
	}
	uri, ok := urim[dirType]
	if !ok {
		common.Logger.Errorf("%s is not a supported director type: [fe, be]", dirType)
		return nil
	}

	payload := map[string]string{"symmetrixId": pm.SymmID}
	return pm.listResourceIDs(uri, payload)
}

// ListPortIDs list FE/BE ports
func (pm PowerMax) ListPortIDs(dirType, dirID string) []string {
	common.Logger.Infof("List director ports for %s:%s:%s", pm.SymmID, dirType, dirID)

	urim := map[string]string{
		"fe": "/performance/FEPort/keys",
		"be": "/performance/BEPort/keys",
	}
	uri, ok := urim[dirType]
	if !ok {
		common.Logger.Errorf("%s is not a supported director type: [fe, be]", dirType)
		return nil
	}

	payload := map[string]string{"symmetrixId": pm.SymmID, "directorId": dirID}
	return pm.listResourceIDs(uri, payload)
}

// GetLatestPortMetrics get the latest FE/BE port metrics
func (pm PowerMax) GetLatestPortMetrics(startTime, endTime int64, dirType, dirID, portID string) map[string]float64 {
	common.Logger.Infof("Get latest metrics for port %s:%s:%s:%s", pm.SymmID, dirType, dirID, portID)

	urim := map[string]string{
		"fe": "/performance/FEPort/metrics",
		"be": "/performance/BEPort/metrics",
	}
	uri, ok := urim[dirType]
	if !ok {
		common.Logger.Errorf("%s is not a supported director type: [fe, be]", dirType)
		return nil
	}

	payload := struct {
		basePayload
		DirectorID string `json:"directorId"`
		PortID     string `json:"portId"`
	}{
		basePayload: basePayload{
			SymmetrixID: pm.SymmID,
			DataFormat:  "Average",
			Metrics:     Metrics[fmt.Sprintf("%sport", dirType)],
			StartDate:   startTime,
			EndDate:     endTime,
		},
		DirectorID: dirID,
		PortID:     portID,
	}
	resp, err := pm.Post(uri, payload)
	reportIssues(resp, err)

	return extractLatestMetrics(resp)
}

// ListStorageGroupIDs list storage group IDs
func (pm PowerMax) ListStorageGroupIDs() []string {
	common.Logger.Infof("List storage groups for %s", pm.SymmID)

	uri := "/performance/StorageGroup/keys"
	payload := map[string]string{"symmetrixId": pm.SymmID}
	return pm.listResourceIDs(uri, payload)
}

// GetLatestStorageGroupMetrics get the latest storage group metrics
func (pm PowerMax) GetLatestStorageGroupMetrics(startTime, endTime int64, id string) map[string]float64 {
	common.Logger.Infof("Get latest storage group metrics for %s:%s from %d to %d", pm.SymmID, id, startTime, endTime)

	uri := "/performance/StorageGroup/metrics"
	payload := struct {
		basePayload
		StorageGroupID string `json:"storageGroupId"`
	}{
		basePayload: basePayload{
			SymmetrixID: pm.SymmID,
			DataFormat:  "Average",
			Metrics:     Metrics["storagegroup"],
			StartDate:   startTime,
			EndDate:     endTime,
		},
		StorageGroupID: id,
	}
	resp, err := pm.Post(uri, payload)
	reportIssues(resp, err)

	return extractLatestMetrics(resp)
}

func (pm PowerMax) listResourceIDs(uri string, payload map[string]string) []string {
	common.Logger.Infof("List resources for %s at uri %s", pm.SymmID, uri)

	resp, err := pm.Post(uri, payload)
	if err != nil || resp.IsError() {
		reportIssues(resp, err)
		return nil
	}

	var data map[string]interface{}
	err = json.Unmarshal(resp.Body(), &data)
	if err != nil {
		common.Logger.Error("Fail to decode HTTP response body")
		return nil
	}

	if len(data) != 1 {
		common.Logger.Error("Cannot understand the resource format")
		return nil
	}

	var ret []string
	for _, info := range data {
		resources := info.([]interface{})
		for _, entry := range resources {
			res := entry.(map[string]interface{})
			for _, v := range res {
				switch v.(type) {
				case string:
					ret = append(ret, v.(string))
				default:
					// Ignore non string fields
				}
			}
		}
	}
	common.Logger.Debugf("Get resources %+v", ret)
	return ret
}

func (pm PowerMax) url(uri string) string {
	url := fmt.Sprintf("%s%s", pm.APIEndPoint, uri)
	common.Logger.Debugf("Full URL: %s", url)
	return url
}

// Get PowerMax HTTP GET encapsulation
func (pm PowerMax) Get(uri string, params map[string]string) (*resty.Response, error) {
	common.ReqCounter <- 1
	resp, err := get(pm.client, pm.url(uri), params)
	<-common.ReqCounter
	return resp, err
}

// Post PowerMax HTTP POST encapsulation
func (pm PowerMax) Post(uri string, body interface{}) (*resty.Response, error) {
	common.ReqCounter <- 1
	resp, err := post(pm.client, pm.url(uri), body)
	<-common.ReqCounter
	return resp, err
}

func newClient(user, password string) *resty.Client {
	client := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).SetBasicAuth(user, password)
	client = client.SetHeader("Content-Type", "application/json").SetHeader("Accept", "application/json")
	return client
}

func get(client *resty.Client, url string, params map[string]string) (*resty.Response, error) {
	common.Logger.Infof("Get %s", url)
	common.Logger.Tracef("Get %s with params %+v", url, params)
	req := client.R().SetQueryParams(params)
	return req.Get(url)
}

func post(client *resty.Client, url string, body interface{}) (*resty.Response, error) {
	common.Logger.Infof("Post to %s", url)
	common.Logger.Tracef("Post body %+v to %s", body, url)
	req := client.R()
	if body != nil {
		return req.SetBody(body).Post(url)
	}
	return req.Post(url)
}

func reportIssues(resp *resty.Response, err error) {
	if err != nil {
		common.Logger.Errorf("HTTP Get error: %s", err.Error())
	}
	if resp != nil && resp.IsError() {
		common.Logger.Errorf("HTTP response error: %+v", resp.RawResponse)
	}
	if resp == nil {
		common.Logger.Error("Empty HTTP response")
	}
}

func extractLatestMetrics(resp *resty.Response) map[string]float64 {
	common.Logger.Debug("Decode response body into metrics")

	if resp == nil {
		common.Logger.Warning("Null response")
		return nil
	}

	if resp.StatusCode() == httpNoContent {
		common.Logger.Warning("HTTP no content")
		return nil
	}

	if resp.StatusCode() == httpPartialContent {
		common.Logger.Warning("HTTP partial content, has not implemented any logic to process such response")
		return nil
	}

	var data baseResult
	err := json.Unmarshal(resp.Body(), &data)
	if err != nil {
		common.Logger.Errorf("Fail to extract metrics from response body: %s", err.Error())
		return nil
	}

	if data.Count == 0 {
		common.Logger.Warning("No metric is available")
		return nil
	}
	common.Logger.Debugf("Response body contains %d x records, the latest will be used", data.Count)

	latest := data.ResultList.Result[data.Count-1].(map[string]interface{})
	ret := map[string]float64{}

	for k, v := range latest {
		// Delete unneeded fields
		if k == "timestamp" {
			continue
		}
		// Cast all value into float64
		switch v.(type) {
		case int:
			ret[k] = float64(v.(int))
		case int32:
			ret[k] = float64(v.(int32))
		case int64:
			ret[k] = float64(v.(int64))
		case float32:
			ret[k] = float64(v.(float32))
		case float64:
			ret[k] = v.(float64)
		default:
			// Delete string values
		}
	}
	return ret
}
