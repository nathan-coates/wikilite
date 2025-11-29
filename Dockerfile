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

# Uncomment and set these to override defaults or enable features
# ENV WIKI_NAME="My Wiki"
# ENV JWKS_URL="https://your-auth0-domain.us.auth0.com/.well-known/jwks.json"
# ENV JWT_ISSUER="https://your-auth0-domain.us.auth0.com/"
# ENV JWT_EMAIL_CLAIM="email"

# Expose the default server port
EXPOSE 8080

VOLUME ["/data"]

CMD ["./wikilite", "serve"]