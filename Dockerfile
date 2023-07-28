FROM golang:1.15 AS build

RUN apt update && apt install -yq go-dep

RUN mkdir -p $GOPATH/src/github.com/Cleafy

WORKDIR $GOPATH/src/github.com/Cleafy/promqueen
COPY . .
RUN dep ensure

WORKDIR $GOPATH/src/github.com/Cleafy/promqueen/bin/promrec
RUN GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o "$GOPATH/bin/promrec" -a -ldflags "-extldflags '-static'" github.com/Cleafy/promqueen/bin/promrec

FROM golang:1.15

WORKDIR /promqueen

COPY --from=build $GOPATH/bin/promrec /promqueen

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y logrotate findutils && rm -rf /var/lib/apt/lists/*

ARG METRICS_DIR="/var/log/cleafy/metrics"

RUN mkdir -p $METRICS_DIR

COPY log_rotate_conf.sh .
COPY rotate-metrics.sh .
COPY entrypoint.sh .

ENTRYPOINT ["./entrypoint.sh"]