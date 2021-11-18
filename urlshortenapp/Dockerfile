FROM golang:1.17-alpine
RUN apk add git
WORKDIR /urlshortenapp

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o /main

# Disable Cgo for tests
ENV CGO_ENABLED 0

EXPOSE 80
CMD ["/main"]
