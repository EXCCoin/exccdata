FROM golang:1.17 as daemon

COPY . /go/src
WORKDIR /go/src/cmd/exccdata
RUN env GO111MODULE=on go build -v

FROM node:lts as gui

WORKDIR /root
COPY ./cmd/exccdata /root
RUN npm install
RUN npm run build

FROM golang:1.17
WORKDIR /
COPY --from=daemon /go/src/cmd/exccdata/exccdata /exccdata
COPY --from=daemon /go/src/cmd/exccdata/views /views
COPY --from=gui /root/public /public

EXPOSE 7777
CMD [ "/exccdata" ]
