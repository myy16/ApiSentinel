FROM golang:1.25-alpine AS builder

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/apisentinel-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && addgroup -S apisentinel && adduser -S apisentinel -G apisentinel
COPY --from=builder /out/apisentinel-server /usr/local/bin/apisentinel-server

USER apisentinel
EXPOSE 3001 50051
ENTRYPOINT ["/usr/local/bin/apisentinel-server"]
