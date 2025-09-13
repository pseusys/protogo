.ONESHELL:
.EXPORT_ALL_VARIABLES:
.DEFAULT_GOAL := help

PROTOGO_LOG_LEVEL := DEBUG
PROTOGO_CACHE := ./temp/

GO_ARGS := version
COMPILER := flatc
COMPILER_ARGS := --version
PROTOGO_ARGS := ${GO_ARGS} -- ${COMPILER} ${PROTOC_ARGS}


build:
	@ # Run 'protogo' locally
	go mod tidy
	go build -o build/protogo .
.PHONY: build

run: build
	@ # Run 'protogo' with given arguments
	./build/protogo ${PROTOGO_ARGS}
.PHONY: run

help: build
	@ # Print 'protogo' help message and exit
	./build/protogo
.PHONY: run

clean:
	@ # Run 'protogo' build artifacts
	rm -f go.sum
	rm -rf build
.PHONY: clean
