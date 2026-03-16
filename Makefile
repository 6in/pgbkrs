BINARY  := pgbackup
CMD     := ./cmd/pgbackup
VERSION ?= dev

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.PHONY: build build-all test test-integration lint clean testdb-start testdb-stop

build:
	go build -o bin/$(BINARY) $(CMD)

build-all:
	@mkdir -p dist
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := dist/$(BINARY)-$(VERSION)-$(OS)-$(ARCH)$(EXT)) \
		echo "Building $(OUT) ..."; \
		GOOS=$(OS) GOARCH=$(ARCH) go build -o $(OUT) $(CMD); \
	)

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
