FROM golang as build

RUN apt update && apt install -yq go-dep

RUN mkdir -p $GOPATH/src/github.com/Cleafy

WORKDIR $GOPATH/src/github.com/Cleafy/promqueen
COPY . .
RUN dep ensure

WORKDIR $GOPATH/src/github.com/Cleafy/promqueen/bin/promrec
RUN GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o "$GOPATH/bin/promrec" -a -ldflags "-extldflags '-static'" github.com/Cleafy/promqueen/bin/promrec

FROM golang:alpine

WORKDIR /promqueen

COPY --from=build $GOPATH/bin/promrec /promqueen

ENTRYPOINT ["/promqueen/promrec"]
