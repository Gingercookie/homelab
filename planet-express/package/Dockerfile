FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o package-service

FROM scratch
COPY --from=builder /app/package-service /
EXPOSE 8080
ENTRYPOINT ["/package-service"]
