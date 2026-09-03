.PHONY: generate fmt vet tidy test cover clean help

help:
	@echo "Valid targets:"
	@echo "  generate - Regenerate wire/v2 Go code from wire/v2/wire.proto via buf"
	@echo "  fmt      - Format code"
	@echo "  vet      - Run go vet static analysis"
	@echo "  tidy     - Tidy go modules"
	@echo "  test     - Run tests with race detection and generate coverage profile"
	@echo "  cover    - View test coverage in browser"
	@echo "  clean    - Remove coverage files"

generate:
	buf generate

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test -v -race -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out

clean:
	rm -f coverage.out