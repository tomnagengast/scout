build:
	go build -o scout .

test:
	go test ./...

fmt:
	gofmt -w *.go
