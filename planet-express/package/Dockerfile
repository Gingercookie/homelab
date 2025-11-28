FROM golang:1.23 as builder
WORKDIR /app
COPY . .
RUN go build -o package-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/package-service /
EXPOSE 8080
ENTRYPOINT ["/package-service"]
