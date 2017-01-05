all: build

PREFIX = gcr.io/google_containers
FLAGS = 

VERSION = v1.3.0-beta.0
GIT_COMMIT := `git rev-parse --short HEAD`

SUPPORTED_KUBE_VERSIONS = "1.4.6"
TEST_NAMESPACE = heapster-e2e-tests

deps:
	which godep || go get github.com/tools/godep

fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs gofmt -s -w

build: clean deps fmt
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -ldflags "-w -X k8s.io/heapster/version.HeapsterVersion=$(VERSION) -X k8s.io/heapster/version.GitCommit=$(GIT_COMMIT)" -o heapster k8s.io/heapster/metrics
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -ldflags "-w -X k8s.io/heapster/version.HeapsterVersion=$(VERSION) -X k8s.io/heapster/version.GitCommit=$(GIT_COMMIT)" -o eventer k8s.io/heapster/events

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean deps sanitize build
	GOOS=linux GOARCH=amd64 godep go test --test.short -race ./... $(FLAGS)

test-unit-cov: clean deps sanitize build
	hooks/coverage.sh

test-integration: clean deps build
	godep go test -v --timeout=60m ./integration/... --vmodule=*=2 $(FLAGS) --namespace=$(TEST_NAMESPACE) --kube_versions=$(SUPPORTED_KUBE_VERSIONS)

container: build
	cp heapster deploy/docker/heapster
	cp eventer deploy/docker/eventer
	docker build -t $(PREFIX)/heapster:$(VERSION) deploy/docker/

grafana:
	docker build -t $(PREFIX)/heapster_grafana:$(VERSION) grafana/

influxdb:
	docker build -t $(PREFIX)/heapster_influxdb:$(VERSION) influxdb/

clean:
	rm -f heapster
	rm -f eventer
	rm -f deploy/docker/heapster
	rm -f deploy/docker/eventer

.PHONY: all deps build sanitize test-unit test-unit-cov test-integration container grafana influxdb clean
