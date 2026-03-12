BINARY := pgbackup
CMD    := ./cmd/pgbackup

.PHONY: build test test-integration lint clean testdb-start testdb-stop

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test -short ./...

test-integration: testdb-start
	@eval $$(./scripts/testdb-start.sh | grep ^export) && go test -p 1 ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not installed, skipping"

clean:
	rm -rf bin/

testdb-start:
	@./scripts/testdb-start.sh

testdb-stop:
	@./scripts/testdb-stop.sh
