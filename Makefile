GITHASH=`git log -1 --pretty=format:"%h" || echo "???"`
CURDATE=`date -u +%Y%m%d%H%M%S`

APPVERSION=${GITHASH}_${CURDATE}

build-race:
	#export $(sed 's/=.*\l\r//' .env)
	go build -v -race -o ./bin/acca-race ./cmd/acca/main.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags "-X main.VERSION=${APPVERSION}" -o ./bin/acca ./cmd/acca/main.go

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
	# protobuf / gRPC
	find ./api -name '*.pb.go' -delete
	protoc --proto_path=. --proto_path=./vendor --govalidators_out=. --gogofast_out=plugins=grpc:. ./api/*.proto

	# reform
	find ./engine -name '*_reform.go' -delete
	go generate ./engine/...
	find ./provider -name '*_reform.go' -delete
	go generate ./provider/...

lint: install
	golangci-lint run ./...

test-short: install
	go test -v -short ./...

test: install
	#export $(sed 's/=.*\l\r//' .env)
	go test -v -count 1 -race -short ./...
	go test -v -count 1 -race -timeout 30m ./tests

up:
	docker-compose up --detach --force-recreate --renew-anon-volumes --remove-orphans

down:
	docker-compose down --volumes --remove-orphans

setup:
	./scripts/pg_exec_files.sh ${PWD}/postgres_schema/*.sql
