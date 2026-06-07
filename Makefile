APP := stock-monitor

.PHONY: run build tidy test clean

run:
	go run ./cmd/server

build:
	mkdir -p bin
	go build -o bin/$(APP) ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...

clean:
	rm -rf bin
