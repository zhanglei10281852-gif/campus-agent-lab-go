FROM node:20-alpine AS frontend-build
WORKDIR /src/frontend-admin
COPY frontend-admin/package.json frontend-admin/package-lock.json ./
RUN npm ci
COPY frontend-admin/ ./
RUN npm run build

FROM golang:1.22.12-alpine AS backend-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/campuslab-server ./cmd/server \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/campuslab-seed ./cmd/seed-user

FROM nginx:1.27-alpine
RUN apk add --no-cache tzdata \
    && addgroup -S campuslab && adduser -S -G campuslab campuslab \
    && mkdir -p /data && chown campuslab:campuslab /data
COPY --from=frontend-build /src/frontend-admin/dist /usr/share/nginx/html
COPY --from=backend-build /out/campuslab-server /usr/local/bin/campuslab-server
COPY --from=backend-build /out/campuslab-seed /usr/local/bin/campuslab-seed
COPY deploy/nginx.fullstack.conf /etc/nginx/conf.d/default.conf
COPY deploy/start-fullstack.sh /usr/local/bin/start-fullstack
RUN sed -i 's/\r$//' /usr/local/bin/start-fullstack \
    && chmod +x /usr/local/bin/start-fullstack
ENV HTTP_ADDR=:8080 \
    DATABASE_PATH=/data/campuslab.db \
    BUSINESS_TIMEZONE=Asia/Shanghai \
    SESSION_TTL=12h
VOLUME ["/data"]
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 CMD wget -q -O /dev/null http://127.0.0.1/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/start-fullstack"]
