.PHONY: generate lint fmt tidy test cover clean help

help:
	@echo "Valid targets:"
	@echo "  generate - Regenerate wire/v2 Go code from wire/v2/wire.proto via buf"
	@echo "  lint     - Format code and run linters"
	@echo "  fmt      - Format code"
	@echo "  tidy     - Tidy go modules"
	@echo "  test     - Run tests with race detection and generate coverage profile"
	@echo "  cover    - View test coverage in browser"
	@echo "  clean    - Remove coverage files"

generate:
	buf generate

lint: fmt
	golangci-lint run

fmt:
	go fmt ./...

tidy:
	go mod tidy

test:
	go test -v -race -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out

clean:
	rm -f coverage.out