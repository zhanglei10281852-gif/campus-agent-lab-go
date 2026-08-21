FROM golang:1.22.12 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags='-s -w' -o /out/campuslab-server ./cmd/server \
    && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags='-s -w' -o /out/campuslab-seed ./cmd/seed-user \
    && mkdir -p /out/data

FROM alpine:3.21
RUN apk add --no-cache tzdata \
    && addgroup -S campuslab && adduser -S -G campuslab campuslab
COPY --from=build /out/campuslab-server /campuslab-server
COPY --from=build /out/campuslab-seed /campuslab-seed
COPY docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=build --chown=campuslab:campuslab /out/data /data
RUN chmod +x /docker-entrypoint.sh
ENV DATABASE_PATH=/data/campuslab.db
USER campuslab
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
