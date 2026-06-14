# Build context is the repository root so the `replace ghnotify/contract =>
# ../../pkg/contract` directive in services/notifications/go.mod can resolve the
# shared contract module. Email templates are embedded into the binary, so no
# HTML assets need to be copied into the runtime image.
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY pkg/contract ./pkg/contract
COPY services/notifications ./services/notifications

WORKDIR /src/services/notifications
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/notifier ./cmd

FROM alpine:latest
LABEL maintainer="vanya04400@gmail.com"

COPY --from=builder /out/notifier /app/notifier
WORKDIR /app

CMD ["./notifier"]
