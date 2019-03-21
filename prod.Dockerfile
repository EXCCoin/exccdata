FROM golang:1.12.1-alpine3.9 as builder

RUN apk add git gcc g++ musl-dev --update --no-cache

WORKDIR /go/src/github.com/EXCCoin/exccdata
COPY . .

ENV GO111MODULE=on
RUN go build -ldflags='-s -w -extldflags "-static"' .


FROM alpine:3.9

WORKDIR /app
COPY --from=builder /go/src/github.com/EXCCoin/exccdata/exccdata .
COPY ./views ./views
COPY ./public ./public

EXPOSE 7777
ENV DATA_DIR=/data
ENV CONFIG_FILE=/app/exccdata.conf
ENV EXCCD_CERT=/app/exccd.cert

CMD ["sh", "-c", "/app/exccdata --appdata=${DATA_DIR} --configfile=${CONFIG_FILE} --exccdcert=${EXCCD_CERT} --logdir=${DATA_DIR}/logs"]
