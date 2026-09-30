

.PHONY: clean all init generate generate_mocks test test_unit test_api

all: build/main

build/main: cmd/main.go generated
	@echo "Building..."
	go build -o $@ $<

clean:
	rm -rf generated

init: clean generate
	go mod tidy
	go mod vendor

# Coverage total excludes generated code (oapi-codegen output, mocks): it is not
# hand-written, so it says nothing about test quality. See spec/design.md §7.1.
test:
	go clean -testcache
	go test -short -coverprofile coverage.out -v ./...
	@grep -v -e '/generated/' -e '\.mock\.gen\.go:' coverage.out > coverage.unit.out
	@echo "Unit coverage (hand-written code):"
	@go tool cover -func coverage.unit.out | tail -1

test_unit: test

test_api:
	go clean -testcache
	go test ./tests/...

generate: generated generate_mocks

generated: api.yml
	@echo "Generating files..."
	mkdir -p generated || true
	oapi-codegen --package generated -generate types,server,spec $< > generated/api.gen.go

INTERFACES_GO_FILES := $(shell find repository -name "interfaces.go")
INTERFACES_GEN_GO_FILES := $(INTERFACES_GO_FILES:%.go=%.mock.gen.go)

generate_mocks: $(INTERFACES_GEN_GO_FILES)
$(INTERFACES_GEN_GO_FILES): %.mock.gen.go: %.go
	@echo "Generating mocks $@ for $<"
	mockgen -source=$< -destination=$@ -package=$(shell basename $(dir $<))