FROM golang:1.26.1 AS build

WORKDIR /usr/src/cleve

RUN curl -Lo /bin/tailwindcss "https://github.com/tailwindlabs/tailwindcss/releases/download/v4.2.2/tailwindcss-linux-x64" && \
	chmod 755 /bin/tailwindcss

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN tailwindcss -i ./assets/css/_input.css -o ./assets/css/style.css
RUN go generate ./...

RUN GOOS=linux CGO_ENABLED=0 go build -v -a -installsuffix cgo -o /bin/cleve ./cmd/cleve 

FROM alpine:3.23 AS final

RUN apk --no-cache add ca-certificates
COPY --from=build /bin/cleve /bin/cleve

CMD [ "cleve", "serve" ]
