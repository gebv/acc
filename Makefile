
build-race:
	go build -v -race -o ./bin/acca-race ./cmd/acca/main.go

run: build-race
	GORACE="halt_on_error=1" ./bin/acca-race
