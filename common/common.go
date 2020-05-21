package common

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

const (
	// MAXREQ Max num. of concurrent requests
	MAXREQ = 100
	// DEBUG log level
	DEBUG = false
	// CFG default config file
	CFG = "config.yml"
)

var (
	// Logger global log object
	Logger *log.Logger
	// ReqCounter control the total num. of concurrent requests
	ReqCounter chan int
	// Config config options
	Config cfgOptions
)

func init() {
	ReqCounter = make(chan int, MAXREQ)

	logInit()
}

func logInit() {
	Logger = log.New()
	Logger.SetOutput(os.Stdout)

	formater := log.TextFormatter{
		FullTimestamp: true,
	}
	Logger.SetFormatter(&formater)

	if DEBUG {
		Logger.SetReportCaller(true)
		Logger.SetLevel(log.TraceLevel)
	} else {
		Logger.SetLevel(log.InfoLevel)
		// Logger.SetLevel(log.WarnLevel)
		// Logger.SetLevel(log.DebugLevel)
		// Logger.SetLevel(log.TraceLevel)
	}
}

type cfgOptions struct {
	PowerMax struct {
		Address  string `yaml:"address"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		SymmID   string `yaml:"symmid"`
	} `yaml:"powermax"`
	Exporter struct {
		Target string `yaml:"target"`
		Update bool   `yaml:"update"`
		Port   int64  `yaml:"port"`
	} `yaml:"exporter"`
}

// CfgInit init config options
func CfgInit() {
	cfg := flag.String("config", CFG, fmt.Sprintf("configuration file, %s as default", CFG))
	flag.Parse()

	Logger.Infof("Use %s as config file", *cfg)

	contents, err := ioutil.ReadFile(*cfg)
	if err != nil {
		Logger.Fatalf("Fail to read config file %s: %s", *cfg, err.Error())
	}

	err = yaml.Unmarshal(contents, &Config)
	if err != nil {
		Logger.Fatalf("Fail to decode config file %s: %s", *cfg, err.Error())
	}
}

// CreateTimeRange create a time range (Unix Epoch time in millisecond) from n x seconds ago to now
func CreateTimeRange(seconds int64) (int64, int64) {
	end := time.Now()
	start := end.Add(time.Duration(-1*seconds) * time.Second)
	return start.Unix() * 1000, end.Unix() * 1000
}
