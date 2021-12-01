FROM golang:alpine

COPY . .

CMD go run ./...