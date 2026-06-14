# Build context is the repository root so the `replace ghnotify/contract =>
# ../../pkg/contract` directive in services/api/go.mod can resolve the shared
# contract module.
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY pkg/contract ./pkg/contract
COPY services/api ./services/api

WORKDIR /src/services/api
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd

FROM alpine:latest
LABEL maintainer="vanya04400@gmail.com"

COPY --from=builder /out/api /app/api
WORKDIR /app

EXPOSE ${SERVER_PORT}

CMD ["./api"]
