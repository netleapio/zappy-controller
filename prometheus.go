package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/netleapio/zappy-framework/protocol"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	gaugeLabels = []string{"device_id", "network"}
	gauges      = initGauges()
)

type PrometheusListener struct {
	network      uint16
	eventChannel chan DeviceChange
	registry     *prometheus.Registry
	manager      *DeviceManager
}

func NewPrometheusListener() *PrometheusListener {
	return &PrometheusListener{
		eventChannel: make(chan DeviceChange, 10),
	}
}

func (l *PrometheusListener) Init(manager *DeviceManager, network uint16) {
	reg := prometheus.NewRegistry()

	for _, g := range gauges {
		reg.MustRegister(g)
	}

	l.network = network
	l.registry = reg
	l.manager = manager
}

func (l *PrometheusListener) Start() {
	// Expose the registered metrics via HTTP.
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(
			l.registry,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics to support exemplars.
				EnableOpenMetrics: true,
			},
		))
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	go func() {
		for {
			change := <-l.eventChannel
			d := l.manager.GetDevice(change.DeviceID)
			if d == nil {
				l.removeDevice(change.DeviceID)
			} else {
				l.updateSensorStats(d)
			}
		}
	}()
}

func (l *PrometheusListener) updateSensorStats(d *DeviceState) {
	labels := l.deviceLabels(d.id)

	for k, v := range d.sensors {
		md := protocol.SensorMetadata[k]
		if md == nil {
			log.Printf("unable to update prometheus, unknown sensor: %v, %v", md, k)
			continue
		}
		gauges[k].With(labels).Set(float64(v) * float64(md.Mult) / float64(md.Div))
	}
}

func (l *PrometheusListener) removeDevice(id uint16) {
	labels := l.deviceLabels(id)

	for _, v := range gauges {
		v.Delete(labels)
	}
}

func (l *PrometheusListener) deviceLabels(id uint16) prometheus.Labels {
	networkStr := strconv.Itoa(int(l.network))
	deviceStr := strconv.Itoa(int(id))
	return prometheus.Labels{"device_id": deviceStr, "network": networkStr}

}

func initGauges() map[protocol.SensorType]*prometheus.GaugeVec {
	result := map[protocol.SensorType]*prometheus.GaugeVec{}

	for t, m := range protocol.SensorMetadata {
		result[t] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "zappy",
			Subsystem: "sensors",
			Name:      m.Name + "_" + m.Unit,
		}, gaugeLabels)
	}

	return result
}
