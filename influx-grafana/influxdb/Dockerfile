# influxdb
FROM ubuntu:trusty

# Install InfluxDB
ENV INFLUXDB_VERSION 0.8.3

RUN apt-get update && apt-get install -y curl && curl -s -o /tmp/influxdb_latest_amd64.deb https://s3.amazonaws.com/influxdb/influxdb_${INFLUXDB_VERSION}_amd64.deb && \
  dpkg -i /tmp/influxdb_latest_amd64.deb && \
  rm /tmp/influxdb_latest_amd64.deb

ADD config.toml /config/config.toml
ADD shardSpace.json /config/shardSpace.json

ADD run.sh /run.sh
RUN chmod +x /run.sh

ENV PRE_CREATE_DB k8s;grafana

EXPOSE 8083 8086 8084 8090 8099

VOLUME ["/data"]

CMD ["/run.sh"]