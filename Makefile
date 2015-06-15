all: build

TAG = v0.14.0
PREFIX = gcr.io/google_containers
FLAGS = 

deps:
	go get github.com/tools/godep
	go get github.com/progrium/go-extpoints

build: clean deps
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go generate github.com/GoogleCloudPlatform/heapster
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -a github.com/GoogleCloudPlatform/heapster/...
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -a github.com/GoogleCloudPlatform/heapster

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean deps sanitize build
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go test --test.short github.com/GoogleCloudPlatform/heapster/... $(FLAGS)

test-unit-cov: clean deps sanitize build
	hooks/coverage.sh

test-integration: clean deps build
	godep go test -v --timeout=30m github.com/GoogleCloudPlatform/heapster/integration/... --vmodule=*=2 $(FLAGS)

container: build
	cp ./heapster ./deploy/docker/heapster
	docker build -t $(PREFIX)/heapster:$(TAG) ./deploy/docker/

.PHONY: grafana
grafana:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) ./grafana/

.PHONY: influxdb
influxdb:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) ./influxdb/

clean:
	rm -f heapster
	rm -f ./extpoints/extpoints.go
	rm -f ./deploy/docker/heapster


