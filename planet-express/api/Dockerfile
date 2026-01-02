FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o planetexpress-api

FROM scratch
COPY --from=builder /app/planetexpress-api /
EXPOSE 8080
ENTRYPOINT ["/planetexpress-api"]
