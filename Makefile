FLAGS?=-v

lint:
	golangci-lint run ./... --timeout 30m -v 

test-race:
	go test $(FLAGS) ./...--race -cover -test.timeout 5s -count 1

test:
	go test $(FLAGS) ./... -cover -test.timeout 5s  -count 1

client:
	go build $(FLAGS) -race -o ./.build/client power/cmd/client 

server:
	go build $(FLAGS) -race -o ./.build/server  power/cmd/server 

gen:
	go generate ./...

mod:
	go mod tidy && go mod vendor

docker-run:
	docker-compose build && docker-compose up 

.PHONY: lint test test-race gen client server docker-run