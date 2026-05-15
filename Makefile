.PHONY: lint

lint:
	golangci-lint fmt
	golangci-lint run

