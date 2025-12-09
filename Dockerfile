FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags "ui,plugins" -o wikilite cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/wikilite .

RUN mkdir -p /data/plugins

# The app will fail to start without a secret. Change this in production!
ENV JWT_SECRET="changeme"

ENV DB_PATH="/data/wiki.db"
ENV LOG_DB_PATH="/data/logs.db"
ENV PLUGIN_PATH="/data/plugins"
ENV PLUGIN_STORAGE_PATH="/data/plugin-storage.db"

# ENV WIKI_NAME="My Wiki"
# ENV INSECURE_COOKIES=true
# ENV TRUST_PROXY_HEADERS=true

EXPOSE 8080

VOLUME ["/data"]

CMD ["./wikilite", "serve"]