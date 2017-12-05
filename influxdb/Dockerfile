FROM golang:GOLANG_VERSION as builder
WORKDIR /go/src/github.com/influxdb/influxdb
RUN curl -sSL https://github.com/influxdata/influxdb/archive/VERSION.tar.gz | tar -xz --strip-components=1
RUN go get github.com/sparrc/gdm 
RUN gdm restore 
RUN ln -s /go/src/github.com/influxdb/influxdb /go/src/github.com/influxdata/ 
RUN GOamd64=ARCH CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags='-s' -o /build/influxd ./cmd/influxd

FROM BASEIMAGE

COPY --from=builder /build/influxd /usr/bin/
COPY config.toml /etc/

ENTRYPOINT ["influxd", "--config", "/etc/config.toml"]
