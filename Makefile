# vi:syntax=make

.ONESHELL:
.DEFAULT_GOAL := help
SHELL := /bin/bash
.SHELLFLAGS = -ec

MAKEFILE_ABSPATH := $(CURDIR)/$(word $(words $(MAKEFILE_LIST)),$(MAKEFILE_LIST))
MAKEFILE_RELPATH := $(call MAKEFILE_ABSPATH)

COMPOSE :=docker-compose -f docker/docker-compose.yml

.PHONY: help
help: ## print help message
	@echo "Usage: make <command>"
	@echo
	@echo "Available commands are:"
	@grep -E '^\S[^:]*:.*?## .*$$' $(MAKEFILE_RELPATH) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-4s\033[36m%-30s\033[0m %s\n", "", $$1, $$2}'
	@echo

.PHONY: build
build: ## build the application
	go build -o wait-for

.PHONY: test
test: ## run unit tests
	go test -race ./...

.PHONY: acceptance-test
acceptance-test: build ## run acceptance tests
	cd test && godog

.PHONY: acceptance-test-docker
acceptance-test-docker: ## run acceptance tests in Docker (if you can't open local ports reliably)
	docker-compose -f test/docker-compose.yml up --build --abort-on-container-exit godog
