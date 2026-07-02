.PHONY: build build-gui test lint run run-gui smoke release clean

BINARY := codelens
GUI_BINARY := codelens-gui
DIR ?= ./testdata/sample_empty
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/codelens

# codelens-gui is the native desktop app (Fyne). Unlike codelens it needs
# CGO plus the system OpenGL/X11 dev headers to build (on Linux:
# pkg-config, libgl1-mesa-dev, xorg-dev or equivalents; macOS/Windows only
# need Xcode / a C compiler, which Fyne's own docs cover). Because it's a
# CGO build it cannot be cross-compiled from one machine like `release`
# does for the CLI — build it natively on/for each target OS.
build-gui:
	go build -ldflags "$(LDFLAGS)" -o bin/$(GUI_BINARY) ./cmd/codelens-gui

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l .)"
	go vet ./...

run:
	go run ./cmd/codelens $(DIR)

run-gui:
	go run ./cmd/codelens-gui

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
	mkdir -p dist
	@for platform in $(PLATFORMS); do \
		goos=$${platform%/*}; \
		goarch=$${platform#*/}; \
		binname=$(BINARY); \
		if [ "$$goos" = "windows" ]; then binname=$(BINARY).exe; fi; \
		tmpdir=$$(mktemp -d); \
		echo "building $$goos/$$goarch"; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go build -ldflags "$(LDFLAGS)" -o "$$tmpdir/$$binname" ./cmd/codelens || { rm -rf "$$tmpdir"; exit 1; }; \
		archive=dist/$(BINARY)-$(VERSION)-$$goos-$$goarch; \
		if [ "$$goos" = "windows" ]; then \
			(cd "$$tmpdir" && zip "$(CURDIR)/$$archive.zip" "$$binname") || { rm -rf "$$tmpdir"; exit 1; }; \
		else \
			tar czf "$$archive.tar.gz" -C "$$tmpdir" "$$binname" || { rm -rf "$$tmpdir"; exit 1; }; \
		fi; \
		rm -rf "$$tmpdir"; \
	done
	@echo "generating checksums"
	@cd dist && (sha256sum *.tar.gz *.zip > checksums.txt 2>/dev/null || shasum -a 256 *.tar.gz *.zip > checksums.txt 2>/dev/null || true)

clean:
	rm -rf bin dist codelens-out
