MAKEFLAGS += --silent

VERSION := $(shell cat VERSION)

%: test

test:
	go test ./... -race -cover

release:
	git tag v$(VERSION)
	git push origin v$(VERSION)
