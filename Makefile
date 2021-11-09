ifdef update
	u=-u
endif

export GO111MODULE=on

all: build

build: fankong_linux.go clean
	go build -ldflags="-s -w" -o bin/fankong

.PHONY: deps
deps:
	go get ${u} -d
	go mod tidy

.PHONY: test
test:
	go test -race ./...

lint:
	golangci-lint run

.PHONY: clean
clean:
	@[ ! -f bin/fankong ] || rm bin/fankong