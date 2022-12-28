init:
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.1

lint:
	golangci-lint run

fmt:
	gofumpt -l -w .
	goimports -l -w .

run:
	go run .

build:
	go build
