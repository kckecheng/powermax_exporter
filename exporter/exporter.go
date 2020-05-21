package exporter

import (
	"strings"
	"sync"

	"github.com/kckecheng/powermax_exporter/common"
	"github.com/kckecheng/powermax_exporter/powermax"
	"github.com/prometheus/client_golang/prometheus"
)

var labelm = map[string][]string{
	"array":        []string{"symmid", "type"},
	"cache":        []string{"symmid", "type", "name"},
	"feport":       []string{"symmid", "type", "director", "port"},
	"beport":       []string{"symmid", "type", "director", "port"},
	"storagegroup": []string{"symmid", "type", "name"},
}

// Exporter Cluster metrics exporter
type Exporter struct {
	resType   string              // Resource type: array, cache, feport, beport, storagegroup
	resources []map[string]string // Available resources
	descs     map[string]*prometheus.Desc
	pm        *powermax.PowerMax
}

// Describe define metric desc
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, name := range powermax.Metrics[e.resType] {
		common.Logger.Infof("Init metric definition for %s: %s", e.resType, name)
		desc := prometheus.NewDesc(name, strings.ReplaceAll(name, "_", " "), labelm[e.resType], nil)
		e.descs[name] = desc
		ch <- desc
	}
}

// Collect logic to collect metrics
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	resType := e.resType
	pm := e.pm
	st, et := common.CreateTimeRange(600) // Collect metrics every 10 minutes to make sure metrics can be always collected

	common.Logger.Infof("Start collecting metrics for %s", resType)
	var wg sync.WaitGroup
	wg.Add(len(e.resources))
	for _, res := range e.resources {
		if resType == "array" {
			go func(symmid, resType string) {
				defer wg.Done()

				metrics := pm.GetLatestArrayMetrics(st, et)
				if metrics == nil || len(metrics) == 0 {
					return
				}

				for k, v := range metrics {
					desc := e.descs[k]
					ch <- prometheus.MustNewConstMetric(
						desc,
						prometheus.GaugeValue,
						v,
						symmid,
						resType,
					)
				}
			}(res["symmid"], res["type"])
		}

		if resType == "cache" {
			go func(symmid, resType, name string) {
				defer wg.Done()

				metrics := pm.GetLatestCacheMetrics(st, et, name)
				if metrics == nil || len(metrics) == 0 {
					return
				}

				for k, v := range metrics {
					desc := e.descs[k]
					ch <- prometheus.MustNewConstMetric(
						desc,
						prometheus.GaugeValue,
						v,
						symmid,
						resType,
						name,
					)
				}
			}(res["symmid"], res["type"], res["name"])
		}

		if resType == "feport" || resType == "beport" {
			go func(symmid, resType, director, port string) {
				defer wg.Done()

				metrics := pm.GetLatestPortMetrics(st, et, resType[:2], director, port)
				if metrics == nil || len(metrics) == 0 {
					return
				}

				for k, v := range metrics {
					desc := e.descs[k]
					ch <- prometheus.MustNewConstMetric(
						desc,
						prometheus.GaugeValue,
						v,
						symmid,
						resType,
						director,
						port,
					)
				}
			}(res["symmid"], res["type"], res["director"], res["port"])
		}

		if resType == "storagegroup" {
			go func(symmid, resType, name string) {
				defer wg.Done()

				metrics := pm.GetLatestStorageGroupMetrics(st, et, name)
				if metrics == nil || len(metrics) == 0 {
					return
				}

				for k, v := range metrics {
					desc := e.descs[k]
					ch <- prometheus.MustNewConstMetric(
						desc,
						prometheus.GaugeValue,
						v,
						symmid,
						resType,
						name,
					)
				}
			}(res["symmid"], res["type"], res["name"])
		}
	}
	wg.Wait()

	if resType == "storagegroup" && common.Config.Exporter.Update {
		common.Logger.Info("Update storage group list in case there are newly created ones")
		sgs := listResources(pm, resType)
		if sgs != nil && len(sgs) > 0 {
			e.resources = sgs
		}
	}
	common.Logger.Infof("Complete collecting metrics for %s", resType)
}

// New init an exporter
func New(pm *powermax.PowerMax, resType string) *Exporter {
	common.Logger.Infof("Init resources dynamically for %s", resType)
	resources := listResources(pm, resType)
	if resources == nil || len(resources) == 0 {
		common.Logger.Fatalf("No %s resource exists", resType)
	}

	e := Exporter{
		resType:   resType,
		resources: resources,
		descs:     map[string]*prometheus.Desc{},
		pm:        pm,
	}

	return &e
}

func listResources(pm *powermax.PowerMax, resType string) []map[string]string {
	ret := []map[string]string{}

	if resType == "array" {
		ret = []map[string]string{map[string]string{"symmid": pm.SymmID, "type": resType}}
	}

	if resType == "cache" {
		cids := pm.ListCachePartitionIDs()
		for _, id := range cids {
			ret = append(ret, map[string]string{"symmid": pm.SymmID, "type": resType, "name": id})
		}
	}

	if resType == "feport" || resType == "beport" {
		dtype := resType[:2]
		dirs := pm.ListDirectorIDs(dtype)
		for _, dir := range dirs {
			ports := pm.ListPortIDs(dtype, dir)
			for _, port := range ports {
				ret = append(ret, map[string]string{"symmid": pm.SymmID, "type": resType, "director": dir, "port": port})
			}
		}
	}

	if resType == "storagegroup" {
		sgs := pm.ListStorageGroupIDs()
		for _, sg := range sgs {
			ret = append(ret, map[string]string{"symmid": pm.SymmID, "type": resType, "name": sg})
		}
	}

	return ret
}
