FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o delivery-simulator

FROM scratch
COPY --from=builder /app/delivery-simulator /
ENTRYPOINT ["/delivery-simulator"]
