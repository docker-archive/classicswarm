TEST?="./..."

build:
	docker build -t swarm .

install:
	godep go install .

fmt:
	gofmt -w=true $(shell find . -type f -name '*.go' -not -path "./Godeps/*")
	goimports -w=true -d $(shell find . -type f -name '*.go' -not -path "./Godeps/*")

test:
	go test -v $(TEST)

race:
	go test -v -race $(TEST)

lint:
	golint ./...

integration:
	test/integration/run.sh

vet:
	@go vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "go vet $(TEST)"
	@go vet $(TEST) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi


.PHONY: fmt test
