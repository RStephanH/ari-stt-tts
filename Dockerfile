FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_NAME=ivr-server

RUN CGO_ENABLED=0 GOOS=linux go build -o  /${APP_NAME} ./cmd/server 

FROM alpine:3.22

RUN adduser -D appuser
USER appuser

WORKDIR /home/appuser

COPY --from=builder /ivr-server ./ivr-server

EXPOSE 8088

ENV ARI_URL="http://localhost:8088/ari" \
  ARI_WS_URL="ws://localhost:8088/ari/events" \
  ARI_USERNAME="asterisk" \
  ARI_PASSWORD="asterisk" \
  ARI_APPLICATION_NAME="app"

ENTRYPOINT [ "./ivr-server" ]
