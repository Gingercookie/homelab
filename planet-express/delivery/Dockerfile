FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o delivery-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/delivery-service /
EXPOSE 8080
ENTRYPOINT ["/delivery-service"]
