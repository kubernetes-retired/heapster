# influxdb
FROM ubuntu:14.04

# Install InfluxDB
ENV INFLUXDB_VERSION 0.8.9

RUN apt-get update && apt-get install -y curl && curl -s -o /tmp/influxdb_latest_amd64.deb http://get.influxdb.org.s3.amazonaws.com/influxdb_${INFLUXDB_VERSION}_amd64.deb && \
  dpkg -i /tmp/influxdb_latest_amd64.deb && \
  rm /tmp/influxdb_latest_amd64.deb

ADD config.toml /config/config.toml
ADD shardSpace.json /config/shardSpace.json

EXPOSE 8083 8086 8084 8090 8099

VOLUME ["/data"]

ENTRYPOINT ["influxdb", "--config", "/config/config.toml"]