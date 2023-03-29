FROM golang:1.19-alpine as daemon
RUN apk add build-base gcc --update --no-cache

COPY . /go/src
WORKDIR /go/src/cmd/exccdata
RUN go build -ldflags='-s -w -extldflags "-static"' .

FROM node:18 as gui

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
