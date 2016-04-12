fmt:
	gofmt -w=true $(shell find . -type f -name '*.go' -not -path "./Godeps/*")
	goimports -w=true -d $(shell find . -type f -name '*.go' -not -path "./Godeps/*")

test:
	go test -v ./...

.PHONY: fmt test
