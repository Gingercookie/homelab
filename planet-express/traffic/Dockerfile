FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o delivery-simulator

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/delivery-simulator /
ENTRYPOINT ["/delivery-simulator"]
