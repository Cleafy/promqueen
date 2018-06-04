package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	cm "github.com/Cleafy/promqueen/model"

	"github.com/mattetti/filebuffer"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/storage/local"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	filetype "gopkg.in/h2non/filetype.v1"
)

var replayType = filetype.NewType("rep", "application/replay")

func replayMatcher(buf []byte) bool {
	header, err := cm.ReadFrameHeader(filebuffer.New(buf))
	if err == nil {
		return cm.CheckVersion(header)
	}
	logrus.Errorf("Malformed frame header!")
	return false
}

var (
	debug             = kingpin.Flag("debug", "Enable debug mode.").Bool()
	error             = kingpin.Flag("error", "Enable error-only mode.").Default("false").Bool()
	nopromcfg         = kingpin.Flag("nopromcfg", "Disable the generation of the prometheus cfg file (prometheus.yml)").Bool()
	dir               = kingpin.Flag("dir", "Input directory.").Short('d').OverrideDefaultFromEnvar("INPUT_DIRECTORY").Default(".").String()
	memoryChunk       = kingpin.Flag("memoryChunk", "Maximum number of chunks in memory").Default("100000000").Int()
	maxChunkToPersist = kingpin.Flag("mexChunkToPersist", "Maximum number of chunks waiting, in memory, to be written on the disk").Default("100000000").Int()
	framereader       = make(<-chan cm.Frame)
	Version           = "unversioned"
	cfgMemoryStorage  = local.MemorySeriesStorageOptions{
		MemoryChunks:       0,
		MaxChunksToPersist: 0,
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
	defer func() {
		if e := recover(); e != nil {
			logrus.Errorf("Frame reader generation failed!, MESSAGE: %v", e)
		}
	}()

	// 1. Check for every file that is GZip or csave format and create the filemap
	files, err := ioutil.ReadDir(*dir)
	if err != nil {
		panic(err)
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
			f, err1 := os.Open(filepath)
			logrus.Debugf("reading header: %v", filepath)
			gz, err2 := gzip.NewReader(f)
			header, err3 := cm.ReadFrameHeader(gz)
			if err1 == nil && err2 == nil && err3 == nil && cm.CheckVersion(header) {
				f.Seek(0, io.SeekStart)
				gz, _ = gzip.NewReader(f)
				readers = append(readers, gz)
			} else {
				if err1 != nil {
					panic(err1)
				}
				if err2 != nil {
					panic(err2)
				}
				if err3 != nil {
					panic(err3)
				}
			}
		}
	}

	framereader = cm.NewMultiReader(readers)
}

func updateURLTimestamp(timestamp int64, name string, url string, body io.Reader) io.Reader {
	dec := expfmt.NewDecoder(body, expfmt.FmtText)
	pr, pw := io.Pipe()
	enc := expfmt.NewEncoder(pw, expfmt.FmtText)
	//ts := timestamp * 1000

	go func() {
		count := 0

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

			lpName := "job"
			urlName := "url"

			for _, metric := range metrics.GetMetric() {
				lp := dto.LabelPair{
					Name:  &lpName,
					Value: &name,
				}
				metric.Label = append(metric.Label, &lp)
				urlp := dto.LabelPair{
					Name:  &urlName,
					Value: &url,
				}
				metric.Label = append(metric.Label, &urlp)
			}

			enc.Encode(&metrics)

			count++
		}

		logrus.Printf("%d metrics unmarshalled for %s", count, url)
		pw.Close()
	}()

	return pr
}

func main() {

	kingpin.Version(Version)

	kingpin.Flag("storage.path", "Directory path to create and fill the data store under.").Default("data").StringVar(&cfgMemoryStorage.PersistenceStoragePath)
	kingpin.Flag("storage.retention-period", "Period of time to store data for").Default("360h").DurationVar(&cfgMemoryStorage.PersistenceRetentionPeriod)

	kingpin.Flag("storage.checkpoint-interval", "Period of time to store data for").Default("30m").DurationVar(&cfgMemoryStorage.CheckpointInterval)
	kingpin.Flag("storage.checkpoint-dirty-series-limit", "Period of time to store data for").Default("10000").IntVar(&cfgMemoryStorage.CheckpointDirtySeriesLimit)

	kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		flag.Set("log.level", "debug")
	}

	if *error {
		logrus.SetLevel(logrus.ErrorLevel)
		flag.Set("log.level", "error")
	}

	logrus.Infoln("Prefilling into", cfgMemoryStorage.PersistenceStoragePath)

	cfgMemoryStorage.MaxChunksToPersist = *maxChunkToPersist
	cfgMemoryStorage.MemoryChunks = *memoryChunk

	localStorage := local.NewMemorySeriesStorage(&cfgMemoryStorage)

	sampleAppender := localStorage

	logrus.Infoln("Starting the storage engine")
	if err := localStorage.Start(); err != nil {
		logrus.Errorln("Error opening memory series storage:", err)
		os.Exit(1)
	}
	defer func() {
		if err := localStorage.Stop(); err != nil {
			logrus.Errorln("Error stopping storage:", err)
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
			logrus.Errorf("Errors occured while reading frame %d, MESSAGE: %v", frame.NameString, err)
			continue
		}
		bytesReader := updateURLTimestamp(frame.Header.Timestamp, frame.NameString(), frame.URIString(), response.Body)

		sdec := expfmt.SampleDecoder{
			Dec: expfmt.NewDecoder(bytesReader, expfmt.FmtText),
			Opts: &expfmt.DecodeOptions{
				Timestamp: model.TimeFromUnix(frame.Header.Timestamp),
			},
		}

		decSamples := make(model.Vector, 0, 1)
		tempSamples := make(model.Vector, 0, 1)

		for err := sdec.Decode(&tempSamples); err == nil; err = sdec.Decode(&tempSamples) {
			decSamples = append(decSamples, tempSamples...)
		}

		logrus.Infoln("Ingested", len(decSamples), "metrics")

		for sampleAppender.NeedsThrottling() {
			logrus.Debugln("THROTTLING: Waiting 100ms for appender to be ready for more data")
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
					logrus.WithFields(logrus.Fields{
						"sample": s,
						"error":  err,
					}).Error("Sample discarded")
				case local.ErrDuplicateSampleForTimestamp:
					numDuplicates++
					logrus.WithFields(logrus.Fields{
						"sample": s,
						"error":  err,
					}).Error("Sample discarded")
				default:
					logrus.WithFields(logrus.Fields{
						"sample": s,
						"error":  err,
					}).Error("Sample discarded")
				}
			}
		}
	}
	// Generate the prometheus.yml in case it does not exist
	promcfgpath := cfgMemoryStorage.PersistenceStoragePath + "/../prometheus.yml"
	if _, err := os.Stat(promcfgpath); os.IsNotExist(err) && !*nopromcfg {
		if err = ioutil.WriteFile(promcfgpath, []byte("global: {}"), os.ModeExclusive|0644); err != nil {
			logrus.Error(err)
		}
	}

	logrus.Info("Exiting! :)")
}
