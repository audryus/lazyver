.PHONY: test vuln release

test:
	@go test -v -count=1 ./...

# Pinned govulncheck version so local scans reproduce CI exactly.
# Override with: make vuln GOVULNCHECK_VERSION=vX.Y.Z
GOVULNCHECK_VERSION ?= v1.8.0

# Runs the same vulnerability scan as CI against all packages.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

update:
	@git fetch origin trunk
	@git checkout trunk
	@git pull --ff-only origin trunk

# Usage: make release VERSION=v1.0.0
# Tags trunk and pushes the tag; pushing the tag triggers the Release
# workflow, which builds the binaries with GoReleaser and publishes
# the GitHub Release. Same as the Manual Release workflow.
release: update test
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=v1.0.0)
endif
	@if ! echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$'; then \
		echo "Invalid version: '$(VERSION)'. Expected format vX.Y.Z"; exit 1; fi
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "Tag $(VERSION) already exists."; exit 1; fi
	@git tag -a "$(VERSION)" -m "Release $(VERSION) (commit $$(git rev-parse --short HEAD))"
	@git push origin "$(VERSION)"
	@echo "Released $(VERSION) — the Release workflow now runs GoReleaser."
