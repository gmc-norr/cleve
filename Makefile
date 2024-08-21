binary_name = cleve
main_path = ./cmd/cleve

.PHONY: cleve
cleve:
	tailwindcss -i ./assets/css/_input.css -o ./assets/css/style.css
	go generate ./...
	mkdir -p bin
	go build -o bin/${binary_name} ${main_path}

.PHONY: test
test:
	go test -v ./...
