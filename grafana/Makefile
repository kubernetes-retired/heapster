
TAG = 2.1.0
PREFIX = kubernetes

all: container

container:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) .

push:
	docker push $(PREFIX)/grafana:$(TAG)


