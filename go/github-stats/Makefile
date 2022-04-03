GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

github-stats:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./github-stats ./cmd/github-stats

clean:
	rm github-stats

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	github-stats \
	clean \
	test \
	lint