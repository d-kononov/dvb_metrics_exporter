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
	"strings"
)

const (
	devDvbPath         = "/dev/dvb/"
	signalSnrMaxNumber = 65535
)

const version string = "v2"

var (
	showVersion     = kingpin.Flag("version", "Print version information").Default().Bool()
	listenAddress   = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9437").String()
	metricsPath     = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	collectInterval = kingpin.Flag("collector.interval", "Interval of collecting metrics from adapters").Default("5s").String()
	snrCorrection   = kingpin.Flag("collector.snr.correction", "Can be > 1, 65535 will be divided by this number to correct SNR value").Default("1").Int()
	ignoreAdapters  = kingpin.Flag("collector.adapter.ignore", "Ignore adapters list, example: 7,8,9").Default("").String()
	apiV5Force      = kingpin.Flag("collector.apiv5force", "Force API v5").Default("false").Bool()
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
	ignoreAdapterList := strings.Split(*ignoreAdapters, ",")
	for _, ignoreAdapter := range ignoreAdapterList {
		for index := 0; index < len(adapters); index++ {
			adapter := adapters[index]
			if adapter.Name() == fmt.Sprintf("adapter%s", ignoreAdapter) {
				log.Debugf("excluding %s adapter from adapter list", adapter.Name())
				// removing item from array (annotation)
				adapters[index] = adapters[len(adapters)-1]
				adapters = adapters[:len(adapters)-1]
			}
		}
	}
	if len(adapters) < 1 {
		log.Fatal("there are no adapters in the list")
	}

	dvbMetricsCollector := &dvbCollector{adapters: adapters}
	go dvbMetricsCollector.startCollectMetrics()

	startServer(dvbMetricsCollector)
}

func startServer(dvbCollector *dvbCollector) {
	log.Infof("Starting DVB metrics exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html>
			<head><title>DVB metrics exporter</title></head>
			<body>
			<h1>DVB metrics exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
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
