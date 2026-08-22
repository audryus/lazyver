.PHONY: tag release test

tag:
	@git tag -a v0.0.6 -m "Release 0.0.6"

release:
	@goreleaser release --clean

# Runs the full test suite with a per-package timeout (prevents hung tests
# from blocking CI forever) and the race detector enabled.
test:
	@go test -v -race -timeout 120s ./...
