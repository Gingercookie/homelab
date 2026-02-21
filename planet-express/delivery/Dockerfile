FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY delivery/ ./delivery/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o delivery-service ./delivery

FROM scratch
COPY --from=builder /app/delivery-service /
EXPOSE 8080
ENTRYPOINT ["/delivery-service"]
