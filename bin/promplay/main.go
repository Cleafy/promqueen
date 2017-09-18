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

	cm "github.com/cleafy/promqueen/model"
	"github.com/mattetti/filebuffer"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/storage/local"
	"github.com/ropes/go-linker-vars-example/src/version"
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
	nopromcfg        = kingpin.Flag("nopromcfg", "Disable the generation of the prometheus cfg file (prometheus.yml)").Bool()
	dir              = kingpin.Flag("dir", "Input directory.").Short('d').OverrideDefaultFromEnvar("INPUT_DIRECTORY").Default(".").String()
	framereader      = make(<-chan cm.Frame)
	Version          = version.GitTag
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

func updateURLTimestamp(timestamp int64, name string, url string, body io.Reader) io.Reader {
	dec := expfmt.NewDecoder(body, expfmt.FmtText)
	pr, pw := io.Pipe()
	//ts := timestamp * 1000

	go func() {
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

			enc := expfmt.NewEncoder(pw, expfmt.FmtText)

			enc.Encode(&metrics)
		}
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
		bytesReader := updateURLTimestamp(frame.Header.Timestamp, frame.NameString(), frame.URIString(), response.Body)

		sdec := expfmt.SampleDecoder{
			Dec: expfmt.NewDecoder(bytesReader, expfmt.FmtText),
			Opts: &expfmt.DecodeOptions{
				Timestamp: model.TimeFromUnix(frame.Header.Timestamp),
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
	// Generate the prometheus.yml in case it does not exist
	promcfgpath := cfgMemoryStorage.PersistenceStoragePath + "/../prometheus.yml"
	if _, err := os.Stat(promcfgpath); os.IsNotExist(err) && !*nopromcfg {
		if err = ioutil.WriteFile(promcfgpath, []byte("global: {}"), os.ModeExclusive|0644); err != nil {
			logrus.Error(err)
		}
	}

	logrus.Info("Exiting! :)")
}
