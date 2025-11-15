FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_NAME=ivr-server

RUN CGO_ENABLED=0 GOOS=linux go build -o  /${APP_NAME} ./

FROM alpine:3.22

RUN adduser -D appuser
USER appuser

WORKDIR /home/appuser

COPY --from=builder /${APP_NAME} ./${APP_NAME}

EXPOSE 8088

ENV ARI_URL="http://172.17.0.1:8088/ari" \
  ARI_WS_URL="ws://172.17.0.1:8088/ari/events" \
  ARI_USERNAME="asterisk" \
  ARI_PASSWORD="asterisk" \
  ARI_APPLICATION_NAME="app"

ENTRYPOINT [ "./ivr-server" ]
