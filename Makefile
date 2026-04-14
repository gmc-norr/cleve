main_path = ./cmd/cleve

.PHONY: all tidy linux test tailwind generate

all: linux

tidy:
	go mod tidy

tailwind:
	tailwindcss -i ./assets/css/_input.css -o ./assets/css/style.css

generate:
	go generate ./...

linux: tidy tailwind generate
	mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/cleve-linux-amd64 ${main_path}

test:
	go test -v ./...
