# vi:syntax=make

.ONESHELL:
.DEFAULT_GOAL := help
SHELL := /bin/bash
.SHELLFLAGS = -ec

TMP_DIR?=./tmp
BASE_DIR=$(shell pwd)
MAKEFILE_ABSPATH := $(CURDIR)/$(word $(words $(MAKEFILE_LIST)),$(MAKEFILE_LIST))
MAKEFILE_RELPATH := $(call MAKEFILE_ABSPATH)

export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export PATH := $(BASE_DIR)/bin:$(PATH)

.PHONY: help
help: ## print help message
	@echo "Usage: make <command>"
	@echo
	@echo "Available commands are:"
	@grep -E '^\S[^:]*:.*?## .*$$' $(MAKEFILE_RELPATH) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-4s\033[36m%-30s\033[0m %s\n", "", $$1, $$2}'
	@echo

.PHONY: clean
clean:
	rm -f wait-for coverage.txt coverage-merged.txt

.PHONY: clean-deps
clean-deps:
	rm -rf ./bin
	rm -rf ./tmp

./bin/golangci-lint:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.42.0

./bin/tparse: ./bin ./tmp
	curl -sfL -o ./tmp/tparse.tar.gz https://github.com/mfridman/tparse/releases/download/v0.8.3/tparse_0.8.3_Linux_x86_64.tar.gz
	tar -xf ./tmp/tparse.tar.gz -C ./bin

./bin:
	mkdir -p ./bin

./tmp:
	mkdir -p ./tmp

.PHONY: deps
deps: ./bin/tparse
	go get -v ./...
	go mod tidy

.PHONY: mocks
mocks: ## generate mocks for interfaces
	mockgen -source=waitfor.go -package=waitfor > waitfor_mock_test.go

.PHONY: build
build: ## build the application
	go build -o wait-for ./cmd/wait-for

.PHONY: lint
lint: ## run linting
	golangci-lint run

.PHONY: test
test: ## run unit tests
	go test -race -cover -json ./... | tparse -all

.PHONY: ci-test
ci-test: ## ci target - run tests to generate coverage data
	go test -coverprofile=coverage.txt -covermode=set ./...

.PHONY: acceptance-test
acceptance-test: build ## run acceptance tests
	rm -rf tmp/coverage
	go build -cover -o wait-for ./cmd/wait-for
	mkdir -p tmp/coverage
	cd test && GOCOVERDIR=../tmp/coverage godog run

.PHONY: acceptance-test-docker
acceptance-test-docker: ## run acceptance tests in Docker (if you can't open local ports reliably)
	rm -rf tmp/coverage
	mkdir -p ./tmp/coverage
	docker-compose -f test/docker-compose.yml up --build --abort-on-container-exit godog

.PHONY: coverage-report
coverage-report: ## collate the coverage data
	mkdir -p tmp/merged
	go tool covdata merge -i=./tmp/coverage,./test/tmp/coverage -o tmp/merged
	go tool covdata textfmt -i=tmp/merged -o coverage-merged.txt
