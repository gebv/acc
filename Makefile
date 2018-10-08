# TODO: setup PGPASSWORD

setup-schema:
	PGPASSWORD=acca  psql -U acca -q -h 127.0.0.1 -d acca -U acca -v ON_ERROR_STOP=1 -f ./schema.sql
.PHONY: setup-schema

setup-functions:
	PGPASSWORD=acca  psql -U acca -q -h 127.0.0.1 -d acca -U acca -v ON_ERROR_STOP=1 -f ./functions.sql
.PHONY: setup-functions

setup: setup-schema setup-functions
.PHONY: setup

install:
	go install -v ./...
	go test -i ./...
.PHONY: install

restart-dev-infra:
	docker-compose down
	docker-compose up -d
	sleep 2
.PHONY: restart-dev-infra

test: install restart-dev-infra setup
	go test -v -count 1 -race -timeout 5m ./tests

	# docker-compose down
.PHONY: test
