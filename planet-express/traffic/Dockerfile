FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY traffic/ ./traffic/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o delivery-simulator ./traffic

FROM scratch
COPY --from=builder /app/delivery-simulator /
ENTRYPOINT ["/delivery-simulator"]
