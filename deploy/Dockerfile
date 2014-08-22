FROM ubuntu:14.10
MAINTAINER vishnuk@google.com

# NOTE: Run build.sh before building this Docker image.

ADD heapster /usr/bin/heapster

ENTRYPOINT ["/usr/bin/heapster"] 
CMD ["-log_dir", "/"]
