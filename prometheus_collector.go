package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"sync"
)

const prefix = "dvb_"

var (
	labelNames  = []string{"adapter"}
	signal      = prometheus.NewDesc(prefix+"signal", "Signal", labelNames, nil)
	snr         = prometheus.NewDesc(prefix+"snr", "SNR", labelNames, nil)
	ber         = prometheus.NewDesc(prefix+"ber", "BER", labelNames, nil)
	lock        = prometheus.NewDesc(prefix+"lock", "LOCK", labelNames, nil)
	versionDesc = prometheus.NewDesc(prefix+"up", "DVB exporter version", nil, prometheus.Labels{"version": version})
	mutex       = &sync.Mutex{}
)

func (c *dvbCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- signal
	ch <- snr
	ch <- ber
	ch <- lock
	ch <- versionDesc
}

func (c *dvbCollector) Collect(ch chan<- prometheus.Metric) {
	mutex.Lock()
	defer mutex.Unlock()

	ch <- prometheus.MustNewConstMetric(versionDesc, prometheus.GaugeValue, 1)

	for _, item := range c.adapterInfos {
		lockVal := 0
		if item.lock {
			lockVal = 1
		}
		l := []string{strconv.Itoa(item.num)}
		ch <- prometheus.MustNewConstMetric(signal, prometheus.GaugeValue, float64(item.signal), l...)
		ch <- prometheus.MustNewConstMetric(snr, prometheus.GaugeValue, float64(item.snr), l...)
		ch <- prometheus.MustNewConstMetric(ber, prometheus.GaugeValue, float64(item.ber), l...)
		ch <- prometheus.MustNewConstMetric(lock, prometheus.GaugeValue, float64(lockVal), l...)
	}
}
