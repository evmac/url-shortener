FROM golang:1.17-alpine
RUN apk add git
WORKDIR /keygensvc

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o /main

# Disable Cgo for tests
ENV CGO_ENABLED 0

EXPOSE 5000
CMD ["/main"]
