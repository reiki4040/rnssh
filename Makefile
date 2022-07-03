.PHONY: test-goreleaser
test-goreleaser:
	goreleaser --snapshot --skip-publish --rm-dist
