all: build

deps:
	go get github.com/progrium/go-extpoints

build: clean deps
	go generate github.com/GoogleCloudPlatform/heapster
	godep go build -a github.com/GoogleCloudPlatform/heapster

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean deps sanitize build
	godep go test --test.short github.com/GoogleCloudPlatform/heapster/...

test-unit-cov: clean deps sanitize build
	hooks/coverage.sh

test-integration: clean deps build
	godep go test -v --timeout=30m ./... --vmodule=*=2 

container: build
	cp ./heapster ./deploy/docker/heapster
	sudo docker build -t heapster:canary ./deploy/docker/

clean:
	rm -f heapster
	rm -f ./extpoints/extpoints.go
	rm -f ./deploy/docker/heapster

