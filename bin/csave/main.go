package main

import (
	"io"
	"net/http"
	"os"
	"time"

	"../../model"

	"net/http/httputil"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	debug    = kingpin.Flag("debug", "Enable debug mode.").Bool()
	interval = kingpin.Flag("interval", "Timeout waiting for ping.").Default("60s").OverrideDefaultFromEnvar("ACTION_INTERVAL").Short('i').Duration()
	umap     = kingpin.Flag("umap", "stringmap [eg. output.met=http://get.uri:port/uri].").Short('u').StringMap()
	dir      = kingpin.Flag("dir", "Output directory.").Short('d').OverrideDefaultFromEnvar("OUTPUT_DIRECTORY").Default("/tmp").String()
	filemap  = make(map[string]io.WriteSeeker)
	version  = "0.0.1"
)

func writerFor(fname string) (io.WriteSeeker, error) {
	nname := *dir + "/" + fname
	if _, err := os.Stat(nname); !os.IsNotExist(err) && filemap[nname] != nil {
		return filemap[nname], nil
	}

	file, err := os.OpenFile(nname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	filemap[nname] = file
	return filemap[nname], nil
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if len(*umap) <= 0 {
		kingpin.Usage()
		return
	}

	ticker := time.NewTicker(*interval)

	for range ticker.C {
		for fname, url := range *umap {
			writer, err := writerFor(fname)
			if err != nil {
				logrus.Errorf("writeFor failed with %v", err)
				continue
			}

			resp, err := http.Get(url)
			if err != nil {
				logrus.Errorf("http.Get: %v", err)
				continue
			}
			defer resp.Body.Close()

			dump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				logrus.Errorf("httputil.DumpResponse: %v", err)
				continue
			}

			frame := model.NewFrame(dump)

			err = model.WriteFrame(writer, url, frame)
			if err != nil {
				logrus.Errorf("model.WriteFrame failed with %v", err)
				continue
			}
		}
	}
}
