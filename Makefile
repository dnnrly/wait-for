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

.PHONY: docker-alpine
docker-alpine: ## build the alpine3.11 image
	docker build -f docker/alpine/Dockerfile -t wait-for:alpine .

.PHONY: docker-golang
docker-bionic: ## build the bionic image
	docker build -f docker/golang/Dockerfile -t wait-for:golang .

.PHONY: docker-build
docker-build: docker-alpine docker-golang

.PHONY: acceptance-test
acceptance-test: build ## run acceptance tests
	cd test && godog
