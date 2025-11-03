FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod tidy
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/requester .

FROM alpine:latest

COPY --from=builder /app/requester /app/requester
ENTRYPOINT ["/app/requester"]
