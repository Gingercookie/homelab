FROM golang:1.23 as builder
WORKDIR /app
COPY . .
RUN go build -o planetexpress-api

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/planetexpress-api /
EXPOSE 8080
ENTRYPOINT ["/planetexpress-api"]
