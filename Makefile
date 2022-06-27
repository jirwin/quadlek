lint:
	golangci-lint run

protogen:
	buf generate

wiregen:
	cd pkg && wire

.PHONY: lint
