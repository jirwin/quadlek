lint:
	golangci-lint run

protogen:
	buf generate

.PHONY: lint
