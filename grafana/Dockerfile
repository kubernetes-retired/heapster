FROM tutum/nginx
MAINTAINER Vishnu Kannan <vishnuk@google.com>

ENV GRAFANA_VERSION 1.9.1
# Environment variables for HTTP AUTH
ENV HTTP_USER admin
ENV HTTP_PASS **Random**

ENV INFLUXDB_METRICS_URL http://localhost:8086/db/k8s
ENV INFLUXDB_GRAFANA_URL http://localhost:8086/db/grafana
ENV INFLUXDB_USER root
ENV INFLUXDB_PASS root

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y wget pwgen apache2-utils curl && wget http://grafanarel.s3.amazonaws.com/grafana-${GRAFANA_VERSION}.tar.gz -O grafana.tar.gz && tar zxf grafana.tar.gz && rm grafana.tar.gz && rm -rf app && mv grafana-${GRAFANA_VERSION} app 

ADD run.sh /run.sh
ADD set_influx_db.sh /set_influx_db.sh
ADD set_basic_auth.sh /set_basic_auth.sh
ADD set_dashboard.sh /set_dashboard.sh

ADD config.js /app/config.js
RUN chmod +x /*.sh && chmod 0644 /app/config.js 
ADD default /etc/nginx/sites-enabled/default
ADD kubernetes-dashboard.json /app/app/dashboards/kubernetes.json

CMD ["/run.sh"]
