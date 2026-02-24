# syntax=docker/dockerfile:1

FROM golang:1.24.0-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/notification-service ./cmd/notification-service


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/notification-service /notification-service

EXPOSE 9091

USER nonroot:nonroot
ENTRYPOINT ["/notification-service"]
