package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	devDvbPath         = "/dev/dvb/"
	signalSnrMaxNumber = 65535
)

const version string = "v1"

var (
	showVersion     = kingpin.Flag("version", "Print version information").Default().Bool()
	listenAddress   = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9437").String()
	metricsPath     = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	collectInterval = kingpin.Flag("collector.interval", "Interval of collecting metrics from adapters").Default("5s").String()
	snrCorrection   = kingpin.Flag("collector.snr.correction", "Can be > 1, 65535 will be divided by this number to correct SNR value").Default("1").Int()
	debug           = kingpin.Flag("collector.debug", "Debug mode").Default("false").Bool()
)

func init() {
	kingpin.Parse()
}

func main() {
	if *showVersion {
		printVersion()
		os.Exit(0)
	}
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	adapters, err := ioutil.ReadDir(devDvbPath)
	if err != nil {
		log.WithError(err).Fatalf("failed to read %s directory", devDvbPath)
	}
	if len(adapters) < 1 {
		log.Fatalf("there are no adapters in %s directory", devDvbPath)
	}

	dvbMetricsCollector := &dvbCollector{adapters: adapters}
	go dvbMetricsCollector.startCollectMetrics()

	startServer(dvbMetricsCollector)
}

func startServer(dvbCollector *dvbCollector) {
	log.Infof("Starting DVB metrics exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DVB metrics exporter"))
		w.WriteHeader(http.StatusOK)
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(dvbCollector)
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      log.New(),
		ErrorHandling: promhttp.ContinueOnError})
	http.Handle(*metricsPath, h)

	log.Infof("Listening for %s on %s", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func printVersion() {
	fmt.Println("dvb-adapter-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Dmitriy Kononov")
}
