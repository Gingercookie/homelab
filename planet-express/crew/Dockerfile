FROM golang:1.23 as builder
WORKDIR /app
COPY . .
RUN go build -o crew-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/crew-service /
EXPOSE 8080
ENTRYPOINT ["/crew-service"]
