BINARY := pgbackup
CMD    := ./cmd/pgbackup

.PHONY: build test lint clean

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not installed, skipping"

clean:
	rm -rf bin/
