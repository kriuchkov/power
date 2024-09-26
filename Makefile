FLAGS?=-v

lint:
	golangci-lint run ./... --timeout 30m -v 

test-race:
	go test $(FLAGS) ./... --race -cover -test.timeout 5s -count 1

test:
	go test $(FLAGS) ./... -cover -test.timeout 5s  -count 1

client:
	go build $(FLAGS) -race -o ./.build/client ./cmd/client 

server:
	go build $(FLAGS) -race -o ./.build/server  ./cmd/server 

mod:
	go mod tidy && go mod vendor

docker-run:
	docker-compose build && docker-compose up 

.PHONY: lint test test-race client server docker-run