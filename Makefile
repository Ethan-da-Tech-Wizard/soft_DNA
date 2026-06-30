.PHONY: build test lint run smoke release clean

BINARY := codelens
DIR ?= ./testdata/sample_empty
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/codelens

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l .)"
	go vet ./...

run:
	go run ./cmd/codelens $(DIR)

smoke: build
	rm -rf codelens-out
	bin/$(BINARY) --version
	bin/$(BINARY) ./testdata/sample_python
	bin/$(BINARY) --ask "what does hello do?" ./testdata/sample_python
	test -f codelens-out/CODEBASE_REPORT.md
	test -f codelens-out/ANSWER.md
	test -f codelens-out/architecture.mmd
	test -f codelens-out/codelens.db

release:
	rm -rf dist
	@for platform in $(PLATFORMS); do \
		goos=$${platform%/*}; \
		goarch=$${platform#*/}; \
		out=dist/$(BINARY)-$(VERSION)-$$goos-$$goarch; \
		if [ "$$goos" = "windows" ]; then out=$$out.exe; fi; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go build -ldflags "$(LDFLAGS)" -o $$out ./cmd/codelens || exit 1; \
	done

clean:
	rm -rf bin dist codelens-out
