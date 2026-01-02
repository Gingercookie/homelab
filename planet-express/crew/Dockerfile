FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o crew-service

FROM scratch
COPY --from=builder /app/crew-service /
EXPOSE 8080
ENTRYPOINT ["/crew-service"]
