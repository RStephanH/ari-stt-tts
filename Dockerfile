FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_NAME=ivr-server

ARG ASTERISK_ID=113
ARG ASTERISK_GUID=112

RUN CGO_ENABLED=0 GOOS=linux go build -o  /${APP_NAME} ./

FROM alpine:3.22

ARG APP_NAME=ivr-server

ARG ASTERISK_ID=113
ARG ASTERISK_GUID=112

RUN addgroup -g ${ASTERISK_GUID} asterisk  && \
  adduser -u ${ASTERISK_ID} -G asterisk -D appuser

RUN mkdir -p /mnt/tts && \
  chown appuser:asterisk /mnt/tts

USER appuser

WORKDIR /home/appuser

COPY --from=builder /${APP_NAME} ./${APP_NAME}

EXPOSE 8088

ENV ARI_URL="http://192.168.122.113:8088/ari" \
  ARI_WS_URL="ws://192.168.122.113:8088/ari/events" \
  ARI_EXTERNAL_MEDIA_BASE_URL="http://192.168.122.113:8088" \
  ARI_IP="192.168.122.113" \
  EXTERNAL_MEDIA_PORT="" \
  ARI_USERNAME="" \
  ARI_PASSWORD="" \
  ARI_APPLICATION_NAME="app" \
  DEEPGRAM_API_KEY="" \
  EXTERNAL_HOST_IP="192.168.122.1" \
  DEEPGRAM_API_KEY=""\
  GEMINI_API_KEY=""

ENTRYPOINT [ "/home/appuser/ivr-server" ]
