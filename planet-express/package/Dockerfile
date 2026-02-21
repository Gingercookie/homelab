FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY package/ ./package/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o package-service ./package

FROM scratch
COPY --from=builder /app/package-service /
EXPOSE 8080
ENTRYPOINT ["/package-service"]
