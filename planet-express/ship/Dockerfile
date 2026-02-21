FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY ship/ ./ship/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o ship-service ./ship

FROM scratch
COPY --from=builder /app/ship-service /
EXPOSE 8080
ENTRYPOINT ["/ship-service"]
