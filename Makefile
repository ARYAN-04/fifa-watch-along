BIN := bin/fifa-hub
DIST_SRC := web/dist
DIST_DST := internal/web/dist

.PHONY: build test run sqlc-generate frontend-build copy-dist all clean

build: copy-dist
	go build -o $(BIN) ./cmd/server

copy-dist: frontend-build
	rm -rf $(DIST_DST)
	cp -R $(DIST_SRC) $(DIST_DST)

frontend-build:
	pnpm --dir web build

test:
	go test ./...

run:
	go run ./cmd/server

sqlc-generate:
	$(shell go env GOPATH)/bin/sqlc generate

all: build test

clean:
	rm -rf $(BIN) $(DIST_DST)
