# No replace, no go.work: inject helpy and infra into GOMODCACHE.
FROM golang:1.26-alpine AS builder
ENV GOMODCACHE=/go/pkg/mod
WORKDIR /app
COPY helpy /go/pkg/mod/github.com/psds-microservice/helpy@v0.1.0
COPY infra /go/pkg/mod/github.com/psds-microservice/infra@v0.0.3
COPY operator-directory-service ./operator-directory-service
WORKDIR /app/operator-directory-service
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /operator-directory-service ./cmd/operator-directory-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /operator-directory-service .
COPY --from=builder /app/operator-directory-service/database ./database
EXPOSE 8098
CMD ["./operator-directory-service", "api"]
