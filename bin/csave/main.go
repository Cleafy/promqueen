package main

import (
	"compress/gzip"
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
	debug      = kingpin.Flag("debug", "Enable debug mode.").Bool()
	enableGZIP = kingpin.Flag("gzip", "Disable gzip mode.").Bool()
	interval   = kingpin.Flag("interval", "Timeout waiting for ping.").Default("60s").OverrideDefaultFromEnvar("ACTION_INTERVAL").Short('i').Duration()
	umap       = kingpin.Flag("umap", "stringmap [eg. service.name=http://get.uri:port/uri].").Short('u').StringMap()
	output     = kingpin.Flag("output", "Output file.").Short('o').OverrideDefaultFromEnvar("OUTPUT_FILE").Default("metrics").String()
	version    = "0.0.1"
	filewriter io.Writer
)

func writerFor() (io.Writer, error) {
	if _, err := os.Stat(*output); !os.IsNotExist(err) && filewriter != nil {
		return filewriter, nil
	}

	file, err := os.OpenFile(*output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	if *enableGZIP {
		filewriter = gzip.NewWriter(file)
	} else {
		filewriter = file
	}
	return filewriter, nil
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
		for sname, url := range *umap {
			writer, err := writerFor()
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

			frame := model.NewFrame(sname, url, dump)

			err = model.WriteFrame(writer, frame)
			if err != nil {
				logrus.Errorf("model.WriteFrame failed with %v", err)
				continue
			}
		}
	}
}
