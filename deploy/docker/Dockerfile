FROM ubuntu
MAINTAINER vishnuk@google.com

# NOTE: Run build.sh before building this Docker image.

# cAdvisor discovery via external files.
RUN mkdir -p /var/run/heapster && touch /var/run/heapster/hosts

ADD heapster /usr/bin/heapster

# Pass in flags to heapster. Options: "--coreos", "--vmodule=*=4", etc.
ENV FLAGS ""

# Select what source sink to use. Options: "influxdb,gcm"
ENV SINK ""

ADD run.sh /run.sh
RUN chmod +x /run.sh

CMD ["/run.sh"]
