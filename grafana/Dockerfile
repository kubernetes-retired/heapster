FROM ubuntu:trusty
MAINTAINER Vishnu Kannan <vishnuk@google.com>

ENV GRAFANA_VERSION 1.9.1

ENV HTTP_USER admin
ENV HTTP_PASS **Random**

ENV INFLUXDB_METRICS_URL http://localhost:8086/db/k8s
ENV INFLUXDB_GRAFANA_URL http://localhost:8086/db/grafana
ENV INFLUXDB_USER root
ENV INFLUXDB_PASS root

RUN apt-get update && \
    apt-get install -y \
        pwgen \
        apache2 \
        apache2-utils \
        curl && \
    apt-get clean && \
    curl -O http://grafanarel.s3.amazonaws.com/grafana-${GRAFANA_VERSION}.tar.gz && \
    tar zxf grafana-${GRAFANA_VERSION}.tar.gz && \
    rm grafana-${GRAFANA_VERSION}.tar.gz && \
    mv grafana-${GRAFANA_VERSION} /usr/share/grafana

EXPOSE 80

ADD run.sh /run.sh
ADD config.js /usr/share/grafana/config.js
ADD kubernetes-dashboard.json /usr/share/grafana/app/dashboards/kubernetes.json
ADD grafana.conf /etc/apache2/sites-available/grafana.conf

RUN chmod +x /run.sh && \
    chmod 0644 /usr/share/grafana/config.js && \
    a2enmod proxy && \
    a2enmod proxy_http && \
    a2dissite 000-default && \
    a2ensite grafana

CMD ["/run.sh"]
