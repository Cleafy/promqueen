package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	cm "cleafy.com/replayer/model"

	"github.com/mattetti/filebuffer"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/storage/local"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	filetype "gopkg.in/h2non/filetype.v1"
)

var replayType = filetype.NewType("rep", "application/replay")

func replayMatcher(buf []byte) bool {
	header, err := cm.ReadFrameHeader(filebuffer.New(buf))
	if err != nil {
		return false
	}
	return cm.CheckVersion(header)
}

var (
	debug            = kingpin.Flag("debug", "Enable debug mode.").Bool()
	dir              = kingpin.Flag("dir", "Input directory.").Short('d').OverrideDefaultFromEnvar("INPUT_DIRECTORY").Default("/tmp").String()
	framereader      = make(<-chan cm.Frame)
	version          = "0.0.1"
	cfgMemoryStorage = local.MemorySeriesStorageOptions{
		MemoryChunks:       1024,
		MaxChunksToPersist: 1024,
		//PersistenceStoragePath:
		//PersistenceRetentionPeriod:
		//CheckpointInterval:         time.Minute*30,
		//CheckpointDirtySeriesLimit: 10000,
		Dirty:          true,
		PedanticChecks: true,
		SyncStrategy:   local.Always,
	}
)

func osfile2fname(fss []os.FileInfo, dir string) []string {
	out := make([]string, len(fss))
	for i, fin := range fss {
		out[i] = dir + "/" + fin.Name()
	}
	return out
}

func generateFramereader() {
	// 1. Check for every file that is GZip or csave format and create the filemap
	files, err := ioutil.ReadDir(*dir)
	if err != nil {
		logrus.Fatalf("generateFilemap: %v", err)
	}
	readers := make([]io.Reader, 0)

	fnames := osfile2fname(files, *dir)
	sort.Sort(cm.ByNumber(fnames))

	logrus.Debugf("fnames: %v", fnames)

	for _, filepath := range fnames {
		logrus.Debugf("filepath: %v", filepath)
		ftype, err := filetype.MatchFile(filepath)
		if err != nil {
			logrus.Debugf("err %v", err)
		}
		if ftype.MIME.Value == "application/replay" {
			f, _ := os.Open(filepath)
			readers = append(readers, f)
		}

		if ftype.MIME.Value == "application/gzip" {
			f, _ := os.Open(filepath)
			logrus.Debugf("reading header: %v", filepath)
			gz, _ := gzip.NewReader(f)
			header, err := cm.ReadFrameHeader(gz)
			if err == nil && cm.CheckVersion(header) {
				f.Seek(0, io.SeekStart)
				gz, _ = gzip.NewReader(f)
				readers = append(readers, gz)
			}
		}
	}

	framereader = cm.NewMultiReader(readers)
}

func updateURLTimestamp(timestamp int64, name string, url string, body io.Reader) []byte {
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

		lpName := "scrapeURL"
		lpValue := "test"

		for _, metric := range metrics.GetMetric() {
			metric.TimestampMs = &timestamp
			lp := dto.LabelPair{
				Name:  &lpName,
				Value: &lpValue,
			}
			metric.Label = append(metric.Label, &lp)
		}

		enc := expfmt.NewEncoder(w, expfmt.FmtText)

		enc.Encode(&metrics)
	}

	return b.Bytes()
}

func main() {
	kingpin.Version(version)

	kingpin.Flag("storage.path", "Directory path to create and fill the data store under.").Default("data").StringVar(&cfgMemoryStorage.PersistenceStoragePath)
	kingpin.Flag("storage.retention-period", "Period of time to store data for").Default("360h").DurationVar(&cfgMemoryStorage.PersistenceRetentionPeriod)

	kingpin.Flag("storage.checkpoint-interval", "Period of time to store data for").Default("30m").DurationVar(&cfgMemoryStorage.CheckpointInterval)
	kingpin.Flag("storage.checkpoint-dirty-series-limit", "Period of time to store data for").Default("10000").IntVar(&cfgMemoryStorage.CheckpointDirtySeriesLimit)

	kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		flag.Set("log.level", "debug")
	}

	log.Infoln("Prefilling into", cfgMemoryStorage.PersistenceStoragePath)
	localStorage := local.NewMemorySeriesStorage(&cfgMemoryStorage)

	sampleAppender := localStorage

	log.Infoln("Starting the storage engine")
	if err := localStorage.Start(); err != nil {
		log.Errorln("Error opening memory series storage:", err)
		os.Exit(1)
	}
	defer func() {
		if err := localStorage.Stop(); err != nil {
			log.Errorln("Error stopping storage:", err)
		}
	}()

	filetype.AddMatcher(replayType, replayMatcher)

	generateFramereader()
	logrus.Debug("frameReader %+v", framereader)

	sout := bufio.NewWriter(os.Stdout)
	defer sout.Flush()

	r := &http.Request{}

	for frame := range framereader {
		response, err := http.ReadResponse(bufio.NewReader(filebuffer.New(frame.Data)), r)
		if err != nil {
			logrus.Error(err)
			return
		}
		bytes := updateURLTimestamp(frame.Header.Timestamp, frame.NameString(), frame.URIString(), response.Body)
		// TODO: here create a prefill nd output them
		//_, err = sout.Write(bytes)
		//if err != nil {
		//	logrus.Error(err)
		//}
		sdec := expfmt.SampleDecoder{
			Dec: expfmt.NewDecoder(filebuffer.New(bytes), expfmt.FmtText),
			Opts: &expfmt.DecodeOptions{
				Timestamp: model.Now(),
			},
		}

		decSamples := make(model.Vector, 0, 1)

		if err := sdec.Decode(&decSamples); err != nil {
			log.Errorln("Could not decode metric:", err)
			continue
		}

		log.Debugln("Ingested", len(decSamples), "metrics")

		for sampleAppender.NeedsThrottling() {
			log.Debugln("Waiting 100ms for appender to be ready for more data")
			time.Sleep(time.Millisecond * 100)
		}

		var (
			numOutOfOrder = 0
			numDuplicates = 0
		)

		for _, s := range model.Samples(decSamples) {
			if err := sampleAppender.Append(s); err != nil {
				switch err {
				case local.ErrOutOfOrderSample:
					numOutOfOrder++
					log.With("sample", s).With("error", err).Info("Sample discarded")
				case local.ErrDuplicateSampleForTimestamp:
					numDuplicates++
					log.With("sample", s).With("error", err).Info("Sample discarded")
				default:
					log.With("sample", s).With("error", err).Info("Sample discarded")
				}
			}
		}
	}
	logrus.Info("Exiting! :)")
}
