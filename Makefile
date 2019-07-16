
build-race:
	go build -v -race -o ./bin/acca-race ./cmd/acca/main.go

run: build-race
	GORACE="halt_on_error=1" ./bin/acca-race

init:
	go install -v ./vendor/github.com/gogo/protobuf/protoc-gen-gogofast
	# go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/gopkg.in/reform.v1/reform
	go install -v ./vendor/github.com/mwitkow/go-proto-validators/protoc-gen-govalidators

install:
	go install -v ./...
	go test -i ./...

gen:
	# reform
	find ./engine -name '*_reform.go' -delete
	go generate ./engine/...

lint: install
	golangci-lint run ./...

setup:
	./scripts/pg_exec_files.sh ${PWD}/postgres_schema/*.sql
