
FROM public.ecr.aws/docker/library/golang:1.26-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/data-sync-job ./cmd/sync-job/main.go

FROM scratch

USER 10001
WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/data-sync-job /app/data-sync-job


CMD ["./data-sync-job"]
