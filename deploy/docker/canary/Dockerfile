FROM golang:1.4.3
MAINTAINER vishnuk@google.com

RUN apt-get install -y git
RUN git clone https://github.com/kubernetes/heapster.git /go/src/k8s.io/heapster

RUN cd /go/src/k8s.io/heapster && make && mv heapster /heapster 

# TODO: Replace the above with:
# RUN cd /go/src/k8s.io/heapster && git checkout -b heapster-scalability remotes/origin/heapster-scalability \
#  && make && mv heapster /heapster && mv eventer /eventer

ENTRYPOINT ["/heapster"]
