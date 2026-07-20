.PHONY: run frontend build test fmt vet check release

run:
	go run ./cmd/storyboard

frontend:
	cd frontend && npm ci && npm run build

build: frontend
	go build -o storyboard ./cmd/storyboard

test:
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

# check runs the same steps a PR should pass before review.
check: fmt vet test

release:
	go run ./scripts/release --version $(VERSION)
