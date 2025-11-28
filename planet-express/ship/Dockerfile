FROM golang:1.23 as builder
WORKDIR /app
COPY . .
RUN go build -o ship-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/ship-service /
EXPOSE 8080
ENTRYPOINT ["/ship-service"]
