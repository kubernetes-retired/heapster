# influxdb

FROM ubuntu

RUN apt-get update
RUN mkdir -p /opt/influxdb/shared/data

ADD http://s3.amazonaws.com/influxdb/influxdb_0.8.0-rc.3_amd64.deb /influxdb_latest_amd64.deb
RUN dpkg -i /influxdb_latest_amd64.deb
RUN rm -rf /opt/influxdb/shared/data

RUN DEBIAN_FRONTEND=noninteractive apt-get install -y curl
ADD run.sh /run.sh
RUN chmod +x /run.sh

ADD config.toml /opt/influxdb/shared/config.toml

ENV DEFAULT_DB k8s

EXPOSE 8083 8086 8084 8090 8099

CMD ["/run.sh"]