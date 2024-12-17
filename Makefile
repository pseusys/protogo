.ONESHELL:
.DEFAULT_GOAL := help

PROTOGO_LOG_LEVEL := DEBUG

GO_ARGS := version
PROTOC_ARGS := --version

PROTOGO_ARGS := ${GO_ARGS} -- ${PROTOC_ARGS}


build:
	@ # Run 'protogo' locally
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
