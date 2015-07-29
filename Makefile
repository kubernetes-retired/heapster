all: build

TAG = v0.17.0
PREFIX = gcr.io/google_containers
FLAGS = 

deps:
	go get github.com/tools/godep
	go get github.com/progrium/go-extpoints

generate:
	go generate

build: clean generate deps
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -a ./...
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -a

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean deps sanitize build
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go test --test.short ./... $(FLAGS)

test-unit-cov: clean deps sanitize build
	hooks/coverage.sh

test-integration: clean deps build
	godep go test -v --timeout=30m ./integration/... --vmodule=*=2 $(FLAGS)

container: build
	cp ./heapster ./deploy/docker/heapster
	docker build -t $(PREFIX)/heapster:$(TAG) ./deploy/docker/

grafana:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) ./grafana/

influxdb:
	docker build -t $(PREFIX)/heapster_influxdb:$(TAG) ./influxdb/

clean:
	rm -f heapster
	rm -f ./extpoints/extpoints.go
	rm -f ./deploy/docker/heapster

.PHONY: all deps build sanitize test-unit test-unit-cov test-integration container grafana influxdb clean
