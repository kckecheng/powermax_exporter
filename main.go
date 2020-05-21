package main

import (
	"fmt"
	"net/http"

	"github.com/kckecheng/powermax_exporter/common"
	"github.com/kckecheng/powermax_exporter/exporter"
	"github.com/kckecheng/powermax_exporter/powermax"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var pm *powermax.PowerMax

func main() {
	common.CfgInit()

	pm = powermax.NewPowerMax(
		common.Config.PowerMax.Address,
		common.Config.PowerMax.User,
		common.Config.PowerMax.Password,
		common.Config.PowerMax.SymmID,
	)
	exporter := exporter.New(pm, common.Config.Exporter.Target)

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter)

	// Add process and go internal stats information
	// reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}), prometheus.NewGoCollector())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "PowerMax Exporter: access /metrics for data")
	})

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	common.Logger.Infof("Start PowerMax Exporter at port %d", common.Config.Exporter.Port)
	common.Logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", common.Config.Exporter.Port), nil))
}
