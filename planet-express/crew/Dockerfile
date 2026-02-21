FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY crew/ ./crew/

RUN CGO_ENABLED=0 GOARCH=arm64 go build -ldflags="-w -s" -o crew-service ./crew

FROM scratch
COPY --from=builder /app/crew-service /
EXPOSE 8080
ENTRYPOINT ["/crew-service"]
