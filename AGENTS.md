# Repository Guidelines

## Project Structure

This repository is a Go module for the Planka API client and CLI. The reusable client, authentication helpers, and their tests are in `pkg/planka/`. The CLI entry point is `cmd/planka/main.go`. `pkg/planka/planka_api.go` is generated from the root `swagger.json`; edit the specification or generation inputs rather than hand-editing generated code. `README.md` documents public usage and development commands.

## Build, Test, and Development

- `go build ./...` compiles the library and CLI packages.
- `go run ./cmd/planka --help` runs the CLI locally and displays its commands.
- `ginkgo run ./pkg/...` runs the Ginkgo/Gomega test suite.
- `go test ./...` runs Go package tests, including the Ginkgo suite bootstrap.
- `go test -cover ./...` reports package test coverage.
- `oapi-codegen --package planka --generate types,client -o pkg/planka/planka_api.go swagger.json` regenerates the API client; use the repository's expected generator version when changing generated output.

## Coding Style

Use standard Go formatting: tabs for indentation and `gofmt` for Go files. Keep package names short and lowercase (`planka`); use exported Go names in `PascalCase` and unexported names in `camelCase`. Keep CLI command behavior in `cmd/planka` and reusable API/auth behavior in `pkg/planka`. Avoid manual edits to generated files.

## Testing

Tests live beside the package code and use Ginkgo/Gomega, with `TestPlanka` registering the suite. Name test files `*_test.go` and describe observable behavior, including HTTP status and error cases. Add focused cases when changing authentication or client requests, then run `ginkgo run ./pkg/...` or `go test ./...`.

## Commits and Pull Requests

Recent commits use short, imperative summaries (for example, “Add OS keychain support for credentials” and “Fix go vet issues and format code”). Follow that style and keep each commit focused. A pull request should explain the user-visible change, note relevant commands run, and link an issue when applicable. Include screenshots only for changes with a visual CLI or documentation impact.

## Configuration and Credentials

The CLI stores credentials in `~/.planka/config.json` and supports OS keychain storage. Do not commit tokens, passwords, or local configuration. Use test credentials or mocked HTTP responses in tests.
