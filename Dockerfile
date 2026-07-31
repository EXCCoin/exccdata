FROM golang:1.25-alpine AS daemon
RUN apk add build-base gcc --update --no-cache

COPY . /go/src
WORKDIR /go/src/cmd/exccdata
ARG APP_PRERELEASE=pre
ARG APP_BUILD=dev
RUN go build -ldflags="-s -w -extldflags \"-static\" -X main.appPreRelease=${APP_PRERELEASE} -X main.appBuild=${APP_BUILD}" .

FROM node:lts AS gui

WORKDIR /root
COPY ./cmd/exccdata /root
RUN npm install
RUN npm run build

FROM alpine:3.17
WORKDIR /app
COPY --from=daemon /go/src/cmd/exccdata/exccdata exccdata
COPY --from=daemon /go/src/cmd/exccdata/views views
COPY --from=gui /root/public public

RUN mkdir /data

EXPOSE 7777
ENV DATA_DIR=/data
ENV CONFIG_FILE=/app/exccdata.conf
CMD ["sh", "-c", "./exccdata --appdata=${DATA_DIR} --configfile=${CONFIG_FILE}"]
