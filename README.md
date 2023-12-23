# PromQueen
**PromQueen** made possible to record _prometheus_ metrics offline.
**PromQueen** can, therefore, backfill the recorded data inside a native _prometheus_ database.

**PromQueen** is composed of two primary tools:

- `promrec` tapes the metrics in a specified output file.
- `promplay` backfills the _prometheus_ database from scratch.

## Build instructions (Linux/OSX)

Clone this repository in your **$GOPATH**:

```
$ mkdir -p $GOPATH/src/github.com/Cleafy
$ cd $GOPATH/src/github.com/Cleafy
$ git clone https://github.com/Cleafy/promqueen.git
$ cd $GOPATH/src/github.com/
$ go mod init
```

Use Go package manager ***dep*** to install the required dependencies:

```
$ cd $GOPATH/src/github.com/Cleafy/promqueen
$ dep ensure
```

To build `promrec`:

```
$ cd $GOPATH/src/github.com/Cleafy/promqueen/bin/promrec
$ go build
```

To build `promplay`:

```
$ cd $GOPATH/src/github.com/Cleafy/promqueen/bin/promplay
$ go build
```

## Usage

### PromREC

```
usage: promrec [<flags>]

Flags:
      --help              Show context-sensitive help (also try --help-long and --help-man).
      --debug             Enable debug mode.
      --gzip              Enable gzip mode.
  -i, --interval=60s      Timeout waiting for ping.
  -u, --umap=UMAP ...     stringmap [eg. service.name=http://get.uri:port/uri].
  -o, --output="metrics"  Output file.
      --version           Show application version.
```

### PromPLAY

```
usage: promplay [<flags>]

Flags:
      --help                 Show context-sensitive help (also try --help-long and --help-man).
      --debug                Enable debug mode. (VERY VERBOSE!)
      --verbose (-v)         Enable info-level message
      --nopromcfg            Disable the generation of the prometheus cfg file (prometheus.yml)
  -d, --dir="/tmp"           Input directory.
      --version              Show application version.
      --storage.path="data"  Directory path to create and fill the data store under.
      --storage.retention-period=360h
                             Period of time to store data for
      --storage.checkpoint-interval=30m
                             Period of time to store data for
      --storage.checkpoint-dirty-series-limit=10000
                             Period of time to store data for
```

### Environment variables

```PROM_ARGS```: The argument for the promqueen service. Output, interval and at least one service is mandatory. 
  - E.g. --output=/var/log/promqueen/metrics/metrics.prom --interval=30s -u serviceName1=URL1 -u serviceName2=URL2 ...


```ROTATION_FILE_LOG```: where the rotation should occurr. Must be the same of the "output" parameter in PROM_ARGS


```ROTATION_PERIOD```: how frequently a rotation will occurr. Default: "daily"

```ROTATION_COUNT```: how many rotation will be retained. Default: 10
 
```ROTATION_SIZE```: how big each rotation file will be in bytes. -1 means no limit. Default: -1. E.g. 100M


```
docker run -d --network=host --name promqueen \
-e ROTATION_FILE_LOG="/var/log/promqueen/metrics/metrics.prom" \
-e PROM_ARGS="--output=/var/log/promqueen/metrics/metrics.prom --interval=30s -u service1=URL1 -u service2=URL2" \
 promqueen_image
```

### Notes

As of today **PromQueen** only supports backfilling inside _prometheus_ local storage. New storage types such as influxdb are not supported.
