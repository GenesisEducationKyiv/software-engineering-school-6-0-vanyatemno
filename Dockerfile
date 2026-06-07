FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY cmd ./cmd
COPY docs ./docs
COPY internal ./internal
COPY go.mod go.sum *.go ./

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/appbin ./cmd/main.go

FROM alpine:latest
LABEL MAINTAINER = <vanya04400@gmail.com>

COPY --from=builder /app /home/appuser/app

WORKDIR /home/appuser/app

EXPOSE ${SERVER_PORT}

CMD ["./appbin"]