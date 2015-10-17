EXAMPLES = examples

.PHONY: format formatted all go-clean pkg-build-install test vet check \
        example-scheduler example-executor

all: go-clean pkg-build-install examples

go-clean:
	go clean

pkg-build-install:
	go install -v ./...

examples: example-scheduler example-executor

example-scheduler:
	rm -rf ${EXAMPLES}/$@
	go build -o ${EXAMPLES}/$@ ${EXAMPLES}/example_scheduler.go

example-executor:
	rm -rf ${EXAMPLES}/$@
	go build -o ${EXAMPLES}/$@ ${EXAMPLES}/example_executor.go

format:
	gofmt -s -w .

formatted:
	! gofmt -s -d . 2>&1 | read diff

vet:
	go vet ./...

test:
	go test -timeout 60s -race ./...

check: formatted vet test
