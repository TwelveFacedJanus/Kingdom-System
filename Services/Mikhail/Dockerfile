FROM golang:tip-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./ 
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /app/server .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 50051

CMD ["/app/server"] 