IMG=golang:1.23.9
BUILD_FLGS= -buildvcs=false -tags netgo -installsuffix netgo -ldflags='-extldflags="static"'
BINS=gwyneth

PKG_NAME=github.com/hinoshiba/gwyneth

CMD_RUN=docker run
CMD_OPT+= --rm --mount type=bind,src=$(CURDIR),dst=/go/src/
CMD_HEAD=/bin/bash -c "cd /go/src &&
CMD_TAIL= && chown $(shell id -u):$(shell id -g) -R ./*"
OUTPUT_PATH=./bin

SRCS := $(shell find . -name '*.go' -type f)

.PHONY: all
all: d-test d-build

.PHONY: d-build
d-build:
	@$(CMD_RUN) $(CMD_OPT) $(IMG) $(CMD_HEAD) make build $(CMD_TAIL)

.PHONY: d-mod
d-mod:
	@$(CMD_RUN) $(CMD_OPT) $(IMG) $(CMD_HEAD) make mod $(CMD_TAIL)

.PHONY: d-modinit
d-modinit:
	@$(CMD_RUN) $(CMD_OPT) $(IMG) $(CMD_HEAD) make modinit $(CMD_TAIL)

.PHONY: d-test
d-test:
	@$(CMD_RUN) $(CMD_OPT) $(IMG) $(CMD_HEAD) make test $(CMD_TAIL)

.PHONY: d-clean
d-clean:
	@$(CMD_RUN) $(CMD_OPT) $(IMG) $(CMD_HEAD) make clean $(CMD_TAIL)

.PHONY: build
build: $(BINS)

$(BINS): $(OUTPUT_PATH) $(SRCS)
	@echo -n "$@ building ..."
	go build $(BUILD_FLGS) -o ./bin/$(@) ./cmd/$(@)
	@echo "done"

.PHONY: mod
mod:
	go mod tidy
	go mod vendor

.PHONY: modinit
modinit:
	go mod init $(PKG_NAME)

.PHONY: test
test:
	go test -v -count=1 -timeout 30s ./...

.PHONY: goclean
clean: ## clean golang
	go clean
	rm -rf $(OUTPUT_PATH)/*

$(OUTPUT_PATH):
	mkdir -p $(OUTPUT_PATH)

.PHONY: help
help: ## help
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {\
		printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF \
	}' $(MAKEFILE_LIST)
