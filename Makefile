
build:
	go build

test:
	go test ./... -count=1

format:
	gofmt -w ./

lint:
	gofmt -d ./
	test -z $(shell gofmt -l ./)
