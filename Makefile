VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BIN     := bin/mr-plow
GOFLAGS := -trimpath

.PHONY: build run test nonet-check vet release universal linux bench fixture clean

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/mr-plow

run: build
	./$(BIN)

test: vet nonet-check
	go test ./...

vet:
	go vet ./...
	GOOS=linux GOARCH=amd64 go vet ./...

# Fails if anything that can open a socket is linked in. (net/url and
# net/netip are pure string parsers pulled in by the ANSI library.)
nonet-check:
	@bad=$$(go list -deps ./cmd/mr-plow | grep -E '^(net|net/http.*|net/rpc.*|net/smtp|net/mail|crypto/tls|os/signal/signal_cgo)$$' || true); \
	if [ -n "$$bad" ]; then echo "network packages linked: $$bad"; exit 1; fi; \
	echo "nonet-check: ok (no socket-capable packages linked)"

# macOS universal binary (arm64 + amd64) and static Linux binaries.
release: universal linux
	@ls -lh dist/

universal:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/mr-plow-darwin-arm64 ./cmd/mr-plow
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/mr-plow-darwin-amd64 ./cmd/mr-plow
	lipo -create -output dist/mr-plow-darwin-universal dist/mr-plow-darwin-arm64 dist/mr-plow-darwin-amd64

linux:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/mr-plow-linux-amd64 ./cmd/mr-plow
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/mr-plow-linux-arm64 ./cmd/mr-plow

bench: build
	./$(BIN) --bench --root $${ROOT:-$$HOME}

# A fake home full of junk to try mr-plow on safely:  make fixture && HOME=/tmp/mr-plow-fixture/home bin/mr-plow
fixture:
	rm -rf /tmp/mr-plow-fixture && scripts/mkfixture.sh /tmp/mr-plow-fixture

clean:
	rm -rf bin dist
