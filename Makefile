# Surat Waris — build & dev tasks
# Catatan: deliverable HARUS bisa dibangun tanpa C compiler (CGO_ENABLED=0).

BINARY      := surat-waris
WIN_BINARY  := siwaris.exe
SQLC        := ./.tools/sqlc.exe
# Versi ditanam ke binary (tampil di footer web). Override: make build-win VERSION=v1.0.1
VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)

.PHONY: dev build build-win build-frontend clean tidy vet generate

## dev: build binary konsol lokal (ada log terminal)
dev:
	go build -o $(WIN_BINARY) .

## build: alias build lokal
build: dev

## build-win: deliverable Windows — tanpa jendela terminal, binary kecil, tanpa CGO
build-win:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w -X main.version=$(VERSION)" -o $(WIN_BINARY) .

## build-frontend: build Svelte -> frontend/dist (Fase C)
build-frontend:
	cd frontend && npm install && npm run build

## generate: jalankan sqlc (typed queries) dari schema.sql + queries.sql
generate:
	$(SQLC) generate

## tidy: rapikan go.mod/go.sum
tidy:
	go mod tidy

## vet: static checks
vet:
	go vet ./...

## clean: hapus artefak build & DB percobaan
clean:
	rm -f $(BINARY) $(WIN_BINARY) surat-waris.db surat-waris.db-shm surat-waris.db-wal
