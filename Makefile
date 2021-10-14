## commom

DATETIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
PACKAGES=$(shell go list ./... | grep -v /vendor/)
VETPACKAGES=$(shell go list ./... | grep -v /vendor/ | grep -v /examples/)
GOFILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
MODFILE=go.mod

all: fmt mod build

.PHONY: fmt vet build

list:
	@echo ${DATETIME}
	@echo ${PACKAGES}
	@echo ${VETPACKAGES}
	@echo ${GOFILES}

fmt:
	@gofmt -s -w ${GOFILES}

init:
	@if [ -f ${MODFILE} ] ; then rm ${MODFILE} ; fi
	@go mod init

mod:
	@go mod tidy

vet:
	@go vet $(VETPACKAGES)

clean:
	@rm bin/logstatd


build:
	@GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/logstatd

imports:
	@goimports -local github.com/yuleihua/logstatd -w .


