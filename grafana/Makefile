
TAG = v3.1.1
PREFIX = kubernetes

all: container

container:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) .

push:
	docker push $(PREFIX)/grafana:$(TAG)


