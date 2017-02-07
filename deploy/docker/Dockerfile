FROM scratch

COPY heapster eventer /
COPY ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/heapster"]
