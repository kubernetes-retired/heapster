FROM gliderlabs/alpine:3.1
MAINTAINER Jimmi Dyson <jimmidyson@gmail.com>
ENTRYPOINT ["/kuisp"]
CMD ["-p", "8080", "-c", "/opt/grafana/config.js.tmpl=/opt/grafana/config.js", "-s", "/db/=${INFLUXDB_PROTO}://${INFLUXDB_HOST}:${INFLUXDB_PORT}/db/"]
EXPOSE 8080

ENV KUISP_VERSION 0.3

ENV GRAFANA_VERSION 1.9.1
ENV INFLUXDB_NAME k8s
ENV GRAFANA_DB_NAME grafana
ENV GRAFANA_DEFAULT_DASHBOARD /dashboard/file/kubernetes.json
ENV INFLUXDB_PROTO http
ENV INFLUXDB_HOST influxdb
ENV INFLUXDB_PORT 8086
ENV INFLUXDB_USER root
ENV INFLUXDB_PASSWORD root
ENV INFLUXDB_EXTERNAL_URL /db/

RUN apk-install ca-certificates curl tar && \
    curl -L https://github.com/jimmidyson/kuisp/releases/download/v${KUISP_VERSION}/kuisp-${KUISP_VERSION}-linux-amd64.tar.gz | \
      tar xzv && \
    mkdir /opt && \
    curl -L http://grafanarel.s3.amazonaws.com/grafana-${GRAFANA_VERSION}.tar.gz | \
      tar xzv -C /opt && \
    mv /opt/grafana-${GRAFANA_VERSION} /opt/grafana && \
    chown nobody:nobody /opt/grafana/ && \
    apk del curl tar

COPY config.js.tmpl /opt/grafana/config.js.tmpl
COPY kubernetes-dashboard.json /opt/grafana/app/dashboards/kubernetes.json

WORKDIR /opt/grafana/
USER nobody
