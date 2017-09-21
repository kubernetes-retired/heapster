FROM scratch

COPY heapster eventer /
COPY ca-certificates.crt /etc/ssl/certs/

# Make sure a GID for a user nobody (65534) is preserved when running in K8s.
COPY passwd /etc/passwd

#   nobody:nobody
USER 65534:65534
ENTRYPOINT ["/heapster"]
