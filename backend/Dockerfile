FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/homelab-api ./cmd/api

FROM alpine:3.23

RUN apk add --no-cache su-exec \
  && addgroup -S homelab \
  && adduser -S homelab -G homelab

WORKDIR /app
COPY --from=build /out/homelab-api /app/homelab-api
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /app/data/photos \
  && chmod +x /app/docker-entrypoint.sh \
  && chown -R homelab:homelab /app

ENV HTTP_ADDR=:8080
ENV DATABASE_PATH=/app/data/homelab.db
ENV PHOTO_STORAGE_DIR=/app/data/photos
ENV ALLOWED_ORIGIN=*

EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/homelab-api"]
