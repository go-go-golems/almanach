.PHONY: build build-web test clean run

BINARY := almanach-render-service

build-web:
	GOWORK=off go run ./cmd/build-web

test:
	GOWORK=off go test ./...

build: build-web
	GOWORK=off go build -tags embed -o ./dist/$(BINARY) ./cmd/almanach-render-service

run:
	GOWORK=off go run ./cmd/almanach-render-service serve

clean:
	rm -rf dist web/dist
