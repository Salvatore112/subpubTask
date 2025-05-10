FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /server /app/server
COPY proto/pubsub.proto /app/proto/pubsub.proto
EXPOSE 50051
ENTRYPOINT ["/app/server"]