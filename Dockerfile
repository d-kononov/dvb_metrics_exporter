FROM golang as builder
RUN go get -d -v github.com/d-kononov/dvb_metrics_exporter
WORKDIR /go/src/github.com/d-kononov/dvb_metrics_exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go/src/github.com/d-kononov/dvb_metrics_exporter/app dvb_exporter
CMD ./dvb_exporter
EXPOSE 9437
