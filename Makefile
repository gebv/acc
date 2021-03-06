# TODO: setup PGPASSWORD

setup-schema:
	PGPASSWORD=acca PGHOST=127.0.0.1 PGDATABASE=acca PGUSER=acca psql -q -v ON_ERROR_STOP=1 -f ./schema.sql

setup-functions:
	PGPASSWORD=acca PGHOST=127.0.0.1 PGDATABASE=acca PGUSER=acca psql -q -v ON_ERROR_STOP=1 -f ./functions.sql

setup-views:
	PGPASSWORD=acca PGHOST=127.0.0.1 PGDATABASE=acca PGUSER=acca psql -q -v ON_ERROR_STOP=1 -f ./views.sql

setup-exts:
	PGPASSWORD=acca PGHOST=127.0.0.1 PGDATABASE=acca PGUSER=acca psql -q -v ON_ERROR_STOP=1 -f ./ext.*.sql

.PHONY: setup
setup: setup-schema setup-functions setup-views setup-exts


init:
	# go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/github.com/gogo/protobuf/protoc-gen-gogo
	go install -v ./vendor/github.com/gogo/protobuf/protoc-gen-gofast

gen:
	# protobuf / gRPC
	find ./api -name '*.pb.go' -delete
	protoc --proto_path=. --proto_path=./vendor --gofast_out=plugins=grpc:. ./api/acca/*.proto

.PHONY: install
install:
	go install -v ./...
	go test -i ./...

restart-dev-infra:
	docker-compose down
	docker-compose up -d
	sleep 10

build-race:
	go build -v -race -o ./bin/acca ./cmd/acca/main.go

.PHONY: test
test: install restart-dev-infra setup
	@echo Unit tests
	go test -v -count 1 -race -timeout 1m ./tests --run=Test0

	@echo Integration tests
	go test -v -count 1 -race -timeout 5m ./tests --grpc-addr=127.0.0.1:3031 --run=Test1

	# docker-compose down

