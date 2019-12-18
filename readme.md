# dvb_exporter
[![Docker Build Statu](https://img.shields.io/docker/build/freeman1988/dvb_metrics_exporter.svg)](https://hub.docker.com/r/freeman1988/dvb_metrics_exporter/builds)
[![Go Report Card](https://goreportcard.com/badge/d-kononov/dvb_metrics_exporter)](https://goreportcard.com/report/github.com/d-kononov/dvb_metrics_exporter)

Prometheus exporter for DVB adapters using https://github.com/ziutek/dvb

This is a simple server that takes all DVB adapters from `/dev/dvb` and scrapes
stats from each adapter and exports them via HTTP for Prometheus consumption.

## Getting Started

### Exported metrics

- `dvb_signal`:  Signal strength in %
- `dvb_snr`:     SNR in %
- `dvb_ber`:     BER - amount of errors
- `dvb_lock`:    Lock (1 or 0)

Each metric has labels `adapter` (adapter number).

Additionally, a `dvb_up` metric reports whether the exporter
is running (and in which version).

### Shell

To run the exporter:

```console
$ ./ping_exporter [options]
```

Help on flags:

```console
$ ./ping_exporter --help
```

Getting the results for testing via cURL:

```console
$ curl http://localhost:9437/metrics
```

### Docker

https://hub.docker.com/r/czerwonk/ping_exporter

To run the ping_exporter as a Docker container, run:

```console
$ docker run -p 9437:9437 -v /dev/dvb/:/dev/dvb/:ro --name dvb_exporter freeman1988/ping_exporter
```

## Contribute

Simply fork and create a pull-request. We'll try to respond in a timely fashion.
