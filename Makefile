all: install-sqlc gen build
.PHONY: all

GOCC?=go

titan-explorer:
	rm -f titan-explorer
	$(GOCC) build $(GOFLAGS) -o titan-explorer .
.PHONY: titan-explorer

install-sqlc:
	go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
.PHONY: install-sqlc

gen:
	sqlc generate
.PHONY: gen

build: titan-explorer
.PHONY: build


gen-swag:
	swag init
.PHONY: gen-swag	