# Build from repo root: docker build -f operator-directory-service/deployments/Dockerfile .
FROM golang:1.21-alpine AS builder
COPY operator-directory-service /app
COPY helpy /helpy
WORKDIR /app
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/operator-directory-service ./cmd/operator-directory-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/operator-directory-service .
EXPOSE 8095
CMD ["./operator-directory-service", "api"]
