.PHONY: build test check

build:
	go build ./cmd/herosiege-server

test:
	go test ./...

check:
	gofmt -w $$(find . -name '*.go' -not -path './.dump/*')
	go vet ./...
	go test ./...
