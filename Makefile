.PHONY: test vet goldens goldens-verify bench fmt

ORACLE := testdata/oracle

test: ## run the parity suite
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

goldens: ## regenerate expectations from pinned juice, then review the diff
	cd $(ORACLE) && npm ci --silent && node generate.mjs

goldens-verify: ## fail if committed goldens disagree with pinned juice
	cd $(ORACLE) && npm ci --silent && node generate.mjs --check

bench-shapes: ## juice across all seven document shapes
	cd $(ORACLE) && npm ci --silent && node shapebench.mjs

bench-node: ## juice on the same document, for comparison
	cd $(ORACLE) && npm ci --silent && node bench.mjs

bench:
	go test -run "^$$" -bench . -benchmem -count 10
