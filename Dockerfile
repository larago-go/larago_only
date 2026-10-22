FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export NODE_EXTRA_CA_CERTS=/run/secrets/proxy_ca; fi; \
    npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.23.11-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -o /campus .

FROM scratch
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY --from=backend /campus /app/campus
COPY --from=frontend /src/frontend/dist /app/static
USER 65532:65532
ENV LISTEN_ADDR=0.0.0.0:8080 STATIC_DIR=/app/static
EXPOSE 8080
ENTRYPOINT ["/app/campus"]
