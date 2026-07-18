.PHONY: run build test fmt vet check

run:
	go run ./cmd/storyboard

build:
	go build -o storyboard ./cmd/storyboard

test:
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

# check runs the same steps a PR should pass before review.
check: fmt vet test
