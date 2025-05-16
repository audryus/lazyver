tag:
	@git tag -a v0.0.6 -m "Release 0.0.6"
release:
	@goreleaser release --clean