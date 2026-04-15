FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app ./cmd/main.go

FROM gcr.io/distroless/static

COPY --from=builder /app /app

ENV STORAGE_DIR=/data/updates

VOLUME /data/updates

EXPOSE 8080

ENTRYPOINT ["/app"]
