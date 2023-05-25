MAKEFLAGS += --silent

VERSION := $(shell cat VERSION)

%: test

test:
	go test ./... -race -cover

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

release:
	git tag v$(VERSION)
	git push origin v$(VERSION)
