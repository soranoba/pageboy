
build:
	go build

test:
	go test ./...

format:
	gofmt -w ./

lint:
	gofmt -d ./
	test -z $(shell gofmt -l ./)
