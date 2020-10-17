FROM golang:1.15.3 as builder

WORKDIR /go/src/github.com/d-kononov/dvb_metrics_exporter

# copy go.mod and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy all go files
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:3.12

COPY --from=builder /go/src/github.com/d-kononov/dvb_metrics_exporter/app /usr/local/bin/dvb-metrics-exporter

CMD dvb-metrics-exporter

EXPOSE 9437
