FROM scratch
MAINTAINER vishnuk@google.com

# NOTE: Run build.sh before building this Docker image.

# Grab cadvisor from the staging directory.
ADD caggregator /usr/bin/caggregator

ENTRYPOINT ["/usr/bin/caggregator"] 
CMD ["-log_dir", "/"]
