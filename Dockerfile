# syntax=docker/dockerfile:1

FROM golang:1.22-bullseye AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/perfugo ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --from=build /bin/perfugo /app/perfugo
COPY web /app/web

ENV SERVER_ADDR=:8080 \
    LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["/app/perfugo"]
