export GO111MODULE = on
export GOPROXY = https://proxy.golang.org

fmt:
	@echo "-> Running go fmt"
	@go fmt ./...

test:
	@echo "-> Running go test"
	@CGO_ENABLED=1 go test -v -race -cover -coverprofile=coverage.out -covermode=atomic ./...

test-integration:
	@echo 'mode: atomic' > coverage.out
	@go list ./... | xargs -n1 -I{} sh -c 'CGO_ENABLED=1 go test -race -tags=integration -covermode=atomic -coverprofile=coverage.tmp -coverpkg $(go list ./... | tr "\n" ",") {} && tail -n +2 coverage.tmp >> coverage.out || exit 255'
	@rm coverage.tmp

test-cover:
	@echo "-> Running go tool cover"
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

bench:
	@echo "-> Running benchmark"
	@go test -v -bench .

profile:
	@echo "-> Running profile"
	@go test -cpuprofile cpu.prof -memprofile mem.prof -v -bench .
