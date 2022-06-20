build:
	docker build .

lint:
	golangci-lint run

.PHONY: build lint
