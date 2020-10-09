
build:
	go build

test:
	go test ./... -count=1
	cd tests; \
		DB=sqlite    go test ./... -count=1 && \
		DB=mysql     go test ./... -count=1 && \
		DB=postgres  go test ./... -count=1 && \
		DB=sqlserver go test ./... -count=1

format:
	gofmt -w ./

lint:
	gofmt -d ./
	test -z $(shell gofmt -l ./)
