FROM ubuntu
MAINTAINER vishnuk@google.com

# NOTE: Run build.sh before building this Docker image.

# cAdvisor discovery via external files.
RUN mkdir -p /var/run/heapster && touch /var/run/heapster/hosts

ADD heapster /usr/bin/heapster

ADD run.sh /run.sh
RUN chmod +x /run.sh

CMD ["/run.sh"]
