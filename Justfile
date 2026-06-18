build:
	go build -o scout ./cmd

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal
