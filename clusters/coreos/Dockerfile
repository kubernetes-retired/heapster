FROM progrium/busybox
MAINTAINER vishnuk@google.com

VOLUME /var/run/heapster
RUN touch /var/run/heapster/hosts

ADD coreos /usr/bin/coreos

ENTRYPOINT ["/usr/bin/coreos"]