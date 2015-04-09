all: build

build: 	
	go generate github.com/GoogleCloudPlatform/heapster
	godep go build -a github.com/GoogleCloudPlatform/heapster

sanitize:
	hooks/check_boilerplate.sh
	hooks/check_gofmt.sh
	hooks/run_vet.sh

test-unit: clean sanitize build 
	godep go test --test.short github.com/GoogleCloudPlatform/heapster/...

test-unit-cov: clean sanitize build
	hooks/coverage.sh	

container: build
	cp ./heapster ./deploy/docker/heapster
	sudo docker build -t heapster:canary ./deploy/docker/

clean:
	rm -f heapster
	rm -f ./deploy/docker/heapster

