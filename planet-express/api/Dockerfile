FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY api/ ./api/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o planetexpress-api ./api

FROM scratch
COPY --from=builder /app/planetexpress-api /
EXPOSE 8080
ENTRYPOINT ["/planetexpress-api"]
