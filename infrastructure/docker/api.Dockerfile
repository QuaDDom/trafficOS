FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps ./apps
COPY packages ./packages
COPY infrastructure/migrations ./infrastructure/migrations
RUN CGO_ENABLED=0 go build -trimpath -o /api ./apps/api && CGO_ENABLED=0 go build -trimpath -o /simulator ./apps/simulator
FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 traffic
USER traffic
WORKDIR /app
COPY --from=build /api /simulator /app/
EXPOSE 8080
ENTRYPOINT ["/app/api"]
