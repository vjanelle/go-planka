.PHONY: all build test coverage fmt vet generate

all: build

build:
	go build ./...

test:
	go tool github.com/onsi/ginkgo/v2/ginkgo run ./pkg/...

coverage:
	go test -cover ./...

fmt:
	gofmt -w cmd/planka/*.go pkg/planka/*.go

vet:
	go vet ./...

generate:
	go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --package planka --generate types,client -o pkg/planka/planka_api.go swagger.json
