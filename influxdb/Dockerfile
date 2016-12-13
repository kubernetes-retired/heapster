FROM ubuntu

MAINTAINER Vishnu kannan "<vishnuk@google.com>"

# Install InfluxDB
ENV INFLUXDB_VERSION 1.1.1

RUN apt-get update && apt-get install -y curl && mkdir /app && curl -s -o /app/influxdb_latest_amd64.deb https://dl.influxdata.com/influxdb/releases/influxdb_${INFLUXDB_VERSION}_amd64.deb && \
  dpkg -i /app/influxdb_latest_amd64.deb && \
  rm /app/influxdb_latest_amd64.deb

ENV PATH=/opt/influxdb:$PATH

ADD config.toml /etc/influxdb.toml

# admin, http, udp, cluster, graphite, opentsdb, collectd
EXPOSE 8083 8086 8086/udp 8088 2003 4242 25826

VOLUME ["/data"]

ENTRYPOINT ["influxd", "--config", "/etc/influxdb.toml"]