package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	labels = []string{"device_id", "network"}

	tempGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zappy",
		Subsystem: "sensors",
		Name:      "temperature_celsius",
	}, labels)
	humidityGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zappy",
		Subsystem: "sensors",
		Name:      "humidity_ratio",
	}, labels)
	pressureGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zappy",
		Subsystem: "sensors",
		Name:      "pressure_pascals",
	}, labels)
	batteryGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "zappy",
		Subsystem: "sensors",
		Name:      "battery_volts",
	}, labels)
)

func startPrometheus() {
	reg := prometheus.NewRegistry()

	// Add Go module build info.
	reg.MustRegister(collectors.NewBuildInfoCollector())
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollections(collectors.GoRuntimeMemStatsCollection | collectors.GoRuntimeMetricsCollection),
	))

	reg.MustRegister(tempGauge)
	reg.MustRegister(humidityGauge)
	reg.MustRegister(pressureGauge)
	reg.MustRegister(batteryGauge)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	fmt.Println("Hello world from new Go Collector!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func prometheusRecord(network uint16, device uint16, tempCelcius, humidityRatio, pressurePascals, batteryVolts float64) {
	networkStr := strconv.Itoa(int(network))
	deviceStr := strconv.Itoa(int(device))

	labels := prometheus.Labels{"device_id": deviceStr, "network": networkStr}

	tempGauge.With(labels).Set(tempCelcius)
	humidityGauge.With(labels).Set(humidityRatio)
	pressureGauge.With(labels).Set(pressurePascals)
	batteryGauge.With(labels).Set(batteryVolts)
}
