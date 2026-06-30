# CodeLens Local

CodeLens Local is an offline codebase explainer. This repository is currently at **M5: local retrieval Q&A**.

Project planning docs live in [`files/`](files/).

## Commands

```sh
make lint
make test
make build
make smoke
bin/codelens --version
bin/codelens ./testdata/sample_empty
bin/codelens ./testdata/sample_python
bin/codelens ./testdata/sample_node
bin/codelens --ask "what does hello do?" ./testdata/sample_python
```

## Release Build

```sh
make release VERSION=v1.0.0
```

Release binaries are written to `dist/` for macOS, Linux, and Windows on amd64/arm64.
