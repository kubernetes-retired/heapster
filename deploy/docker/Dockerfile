FROM progrium/busybox
MAINTAINER vishnuk@google.com

# cAdvisor discovery via external files.
VOLUME /var/run/heapster/hosts

RUN opkg-install ca-certificates
RUN for cert in `ls -1 /etc/ssl/certs/*.crt | grep -v /etc/ssl/certs/ca-certificates.crt`; do cat "$cert" >> /etc/ssl/certs/ca-certificates.crt; done

ADD heapster /heapster
ENTRYPOINT ["/heapster"]
