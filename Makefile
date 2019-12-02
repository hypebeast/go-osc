PKG = ./osc

all: format coverage

help:
	@echo "Usage make [TARGET]"
	@echo "TARGETS:"
	@echo "  all               format, build and run tests"
	@echo "  test              runs all tests"
	@echo "  style             checks the code style"
	@echo "  format            runs go fmt"
	@echo "  vet               vetting code"
	@echo "  lint              runs golint"
	@echo "  coverage          runs the tests and creates a coverage report"

test:
	@echo ">> Running tests"
	@go test -v $(PKG)

coverage:
	@echo ">> Running tests with coverage"
	@go test -v -covermode=count -coverprofile=coverage.out  $(PKG)

style:
	@echo ">> Checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

format:
	@echo ">> Formatting code"
	@gofmt -l -w $(shell find . -path ./vendor -prune -o -name '*.go' -print)

vet:
	@echo ">> Vetting code"
	@go vet $(PKG)

lint:
	@echo ">> Linting code"
	@golint $(PKG)

.PHONY: all test style format vet coverage lint
