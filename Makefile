.PHONY: dev build run db-install test

DB_USER ?= root
DB_NAME ?= goframework

dev:
	air -c .air.toml

build:
	go build -o bin/server cmd/server/main.go

run: build
	./bin/server -config configs/config.prod.yaml

db-install:
	mysql -u $(DB_USER) -p $(DB_NAME) < db/install.sql

test:
	go test ./... -v
