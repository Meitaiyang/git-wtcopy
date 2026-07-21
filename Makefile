BINARY := git-wtcopy

.PHONY: build test vet lint fmt clean

build:
	go build -o $(BINARY) ./cmd/git-wtcopy

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -l .

clean:
	rm -f $(BINARY)
