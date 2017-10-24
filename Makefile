
VERSION=v0.1
GITHASH=`git log -1 --pretty=format:"%h" || echo "???"`
CURDATE=`date -u +%Y%m%d.%H%M%S`
REPO_URL=github.com
ORG_NAME=gebv
REPO_NAME=accounting

APPVERSION=${VERSION}-${GITHASH}:${CURDATE}

include .env
export $(shell sed 's/=.*//' .env)

docker-prebuild:
	docker build -t ${REPO_NAME}-app-builder -f Dockerfile.build .
docker-build: docker-prebuild
	docker run -it --rm --name ${REPO_NAME}-app-make-build \
		-v "${PWD}":/go/src/${REPO_URL}/${ORG_NAME}/${REPO_NAME} \
		-w /go/src/${REPO_URL}/${ORG_NAME}/${REPO_NAME} \
		${REPO_NAME}-app-builder make build

build:
	CGO_ENABLED=0 go build \
			-o bin/app \
			-v \
			-ldflags "-X main.VERSION=${APPVERSION}" \
			-a ./main.go
.PHONY: build

gogenerate:
	go generate ./invoices.go
	go generate ./transactions.go
	go generate ./accounts.go
.PHONY: gogenerate

test:
	go test -v ./
.PHONY: test