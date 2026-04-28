FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wendao-server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/wendao-server /app/wendao-server
COPY config/config.yaml /app/config/config.yaml

RUN mkdir -p /app/uploads /app/log \
    && chown -R app:app /app

USER app

EXPOSE 8089

CMD ["/app/wendao-server"]
