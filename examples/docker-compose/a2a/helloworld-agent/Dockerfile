FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache curl
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o helloworld-agent .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache curl
COPY --from=builder /app/helloworld-agent .
EXPOSE 8080
CMD ["./helloworld-agent"]
