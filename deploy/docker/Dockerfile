FROM progrium/busybox
MAINTAINER vishnuk@google.com

# cAdvisor discovery via external files.
VOLUME /var/run/heapster/hosts

ADD heapster /heapster
ENTRYPOINT ["/heapster"]
