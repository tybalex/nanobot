default: build

GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null | xargs -I {} echo -X 'github.com/nanobot-ai/nanobot/pkg/version.Tag={}')
GO_LD_FLAGS := "-s -w $(GIT_TAG)"
build:
	go build -ldflags=$(GO_LD_FLAGS) -o bin/nanobot .