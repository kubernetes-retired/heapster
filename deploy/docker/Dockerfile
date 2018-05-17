FROM alpine:3.6

COPY heapster eventer /

RUN apk update
RUN apk --no-cache add ca-certificates

# Make sure nobody:nobody is present in the image since Docker needs the GID.
# Otherwise it will default to GID 0 (root) when the container starts with
# the user explicitly specified. This happens with Kubernetes+Docker combo.
RUN grep -q ':65534:65534:' /etc/passwd

#   nobody:nobody
USER 65534:65534
ENTRYPOINT ["/heapster"]
