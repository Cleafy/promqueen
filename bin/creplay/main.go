package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"

	"../../model"

	"github.com/gorilla/mux"
	"github.com/goware/urlx"
	"github.com/mattetti/filebuffer"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	filetype "gopkg.in/h2non/filetype.v1"
)

var replayType = filetype.NewType("rep", "application/replay")

func replayMatcher(buf []byte) bool {
	header, err := model.ReadHeader(filebuffer.New(buf))
	if err != nil {
		return false
	}
	return model.CheckVersion(header)
}

var (
	debug        = kingpin.Flag("debug", "Enable debug mode.").Bool()
	dir          = kingpin.Flag("dir", "Input directory.").Short('d').OverrideDefaultFromEnvar("INPUT_DIRECTORY").Default("/tmp").String()
	framereaders = make(map[string]<-chan model.Frame)
	version      = "0.0.1"
)

func osfile2fname(fss []os.FileInfo, dir string) []string {
	out := make([]string, len(fss))
	for i, fin := range fss {
		out[i] = dir + "/" + fin.Name()
	}
	return out
}

func frameReader2PortNumberMap() map[int16]*mux.Router {
	out := make(map[int16]*mux.Router)
	for uri := range framereaders {
		u, err := urlx.Parse(uri)
		logrus.Debugf("URL2Port url %s", uri)
		if err != nil {
			logrus.Infof("Error parsing URI: %s %v", uri, err)
			continue
		}
		i, err := strconv.ParseInt(u.Port(), 10, 16)
		if err != nil {
			logrus.Debugf("Error parsing PORT: %s %v", u.Port(), err)
			i = 80
		}

		if out[int16(i)] == nil {
			out[int16(i)] = mux.NewRouter()
		}

		path := "/"
		if u.Path != "" {
			logrus.Debugf("path %s", path)
			path = u.Path
		}

		hf := out[int16(i)].HandleFunc(path, handlerGenerator(uri))

		if u.Hostname() != "" {
			logrus.Debugf("host %s", u.Hostname())
			hf.Host(u.Hostname())
		}
	}

	return out
}

func generateFramereaders() {
	filemap := make(map[string][]io.Reader)
	// 1. Check for every file that is GZip or csave format and create the filemap
	files, err := ioutil.ReadDir(*dir)
	if err != nil {
		logrus.Fatalf("generateFilemap: %v", err)
	}

	fnames := osfile2fname(files, *dir)
	sort.Sort(model.ByNumber(fnames))

	logrus.Debugf("fnames: %v", fnames)

	for _, filepath := range fnames {
		logrus.Debugf("filepath: %v", filepath)
		ftype, err := filetype.MatchFile(filepath)
		if err != nil {
			logrus.Debugf("err %v", err)
		}
		if ftype.MIME.Value == "application/replay" {
			f, _ := os.Open(filepath)
			logrus.Debugf("reading header: %v", filepath)
			header, _ := model.ReadHeader(f)
			f.Seek(0, io.SeekStart)
			filemap[header.URIString()] = append(filemap[header.URIString()], f)
		}

		if ftype.MIME.Value == "application/gzip" {
			f, _ := os.Open(filepath)
			logrus.Debugf("reading header: %v", filepath)
			gz, _ := gzip.NewReader(f)
			header, err := model.ReadHeader(gz)
			if err == nil {
				f.Seek(0, io.SeekStart)
				gz, _ = gzip.NewReader(f)
				filemap[header.URIString()] = append(filemap[header.URIString()], gz)
			}
		}
	}

	// 2. generate the framereader channel from the filesmap
	for url, readers := range filemap {
		framereaders[url] = model.NewMultiReader(readers)
	}
}

func updateTimestamp(timestamp int64, body io.Reader) []byte {
	dec := expfmt.NewDecoder(body, expfmt.FmtText)
	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	for {
		var metrics dto.MetricFamily
		err := dec.Decode(&metrics)
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Error(err)
			break
		}

		for _, metric := range metrics.GetMetric() {
			metric.TimestampMs = &timestamp
		}

		enc := expfmt.NewEncoder(w, expfmt.FmtText)

		enc.Encode(&metrics)
	}

	return b.Bytes()
}

func handlerGenerator(url string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		frame := <-framereaders[url]
		response, err := http.ReadResponse(bufio.NewReader(filebuffer.New(frame.Data)), r)
		if err != nil {
			logrus.Error(err)
			return
		}
		w.Write(updateTimestamp(frame.Timestamp, response.Body))
	}
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	filetype.AddMatcher(replayType, replayMatcher)

	generateFramereaders()
	ports := frameReader2PortNumberMap()
	logrus.Debug("frameReader %+v", framereaders)
	logrus.Debug("ports %+v", ports)

	done := make(chan bool)
	for port := range ports {
		go func(port int16) {
			logrus.Infof("Listening on port %d", port)
			logrus.Error(http.ListenAndServe(":"+strconv.FormatInt(int64(port), 10), ports[port]))
			done <- true
		}(port)
	}

	for i := 0; i < len(ports); i++ {
		<-done
	}
	logrus.Info("Exiting! :)")
}
