all: build

TAG = v1.2.0
PREFIX = gcr.io/google_containers
FLAGS = 

SUPPORTED_KUBE_VERSIONS = "1.3.6"
TEST_NAMESPACE = heapster-e2e-tests

verify-glide-installation:
	which glide || go get github.com/Masterminds/glide
	which glide-vc || go get github.com/sgotti/glide-vc

deps: verify-glide-installation
	glide install --strip-vendor
	glide-vc --only-code --no-tests --keep="**/*.json.in"

update-deps: verify-glide-installation
	glide update --strip-vendor
	glide-vc --only-code --no-tests --keep="**/*.json.in"

ifeq ($(wildcard vendor/*),)
build: clean deps
else
build: clean
endif
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o heapster k8s.io/heapster/metrics
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o eventer k8s.io/heapster/events

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean deps sanitize build
	GOOS=linux GOARCH=amd64 go test --test.short -race ./... $(FLAGS)

test-unit-cov: clean deps sanitize build
	hooks/coverage.sh

test-integration: clean deps build
	go test -v --timeout=60m ./integration/... --vmodule=*=2 $(FLAGS) --namespace=$(TEST_NAMESPACE) --kube_versions=$(SUPPORTED_KUBE_VERSIONS)

container: build
	cp heapster deploy/docker/heapster
	cp eventer deploy/docker/eventer
	docker build -t $(PREFIX)/heapster:$(TAG) deploy/docker/

grafana:
	docker build -t $(PREFIX)/heapster_grafana:$(TAG) grafana/

influxdb:
	docker build -t $(PREFIX)/heapster_influxdb:$(TAG) influxdb/

clean:
	rm -f heapster
	rm -f eventer
	rm -f deploy/docker/heapster
	rm -f deploy/docker/eventer

.PHONY: all verify-glide-installation deps update-deps build sanitize test-unit test-unit-cov test-integration container grafana influxdb clean
