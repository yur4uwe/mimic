package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Mountpoint string
	URL        string

	TTL        time.Duration
	MaxEntries int

	Username string
	Password string

	Verbose bool
	StdLog  string
	ErrLog  string
}

func ParseConfig(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	// intermediate struct mirrors config file keys (simple mapping)
	var raw struct {
		Mpoint     string `toml:"mpoint"`
		URL        string `toml:"url"`
		Username   string `toml:"username"`
		Password   string `toml:"password"`
		TTL        string `toml:"ttl"`
		MaxEntries int    `toml:"max-entries"`
		Verbose    bool   `toml:"verbose"`
		Std        string `toml:"std"`
		Err        string `toml:"err"`
	}

	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if raw.Mpoint != "" {
		cfg.Mountpoint = raw.Mpoint
	}
	if raw.URL != "" {
		cfg.URL = raw.URL
	}
	cfg.Username = raw.Username
	cfg.Password = raw.Password

	if raw.TTL != "" {
		if d, err := time.ParseDuration(raw.TTL); err == nil {
			cfg.TTL = d
		}
	}
	if raw.MaxEntries != 0 {
		cfg.MaxEntries = raw.MaxEntries
	}
	cfg.Verbose = raw.Verbose
	cfg.StdLog = raw.Std
	cfg.ErrLog = raw.Err

	return cfg, nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <mountpoint> <server>\n", os.Args[0])
	flag.PrintDefaults()
}

func ParseCommandLineArgs() (*Config, []string, error) {
	var (
		configFPtr    = flag.StringP("config", "c", "", "path to config file")
		userPtr       = flag.StringP("user", "u", "", "username:password (shorthand)")
		ttlPtr        = flag.DurationP("ttl", "t", time.Minute, "cache TTL")
		maxEntriesPtr = flag.IntP("max-entries", "m", 1000, "cache max entries")
		verbosePtr    = flag.BoolP("verbose", "v", false, "enable verbose logging")
		stdlogPtr     = flag.StringP("stdlog", "s", "", "path to standard log file")
		errlogPtr     = flag.StringP("errlog", "e", "", "path to error log file")
	)

	flag.Usage = usage
	flag.Parse()

	fileCfg, err := ParseConfig(*configFPtr)
	if err != nil {
		return nil, nil, err
	}

	cfg := &Config{
		Mountpoint: fileCfg.Mountpoint,
		URL:        fileCfg.URL,
		TTL:        fileCfg.TTL,
		MaxEntries: fileCfg.MaxEntries,
		Username:   fileCfg.Username,
		Password:   fileCfg.Password,
		Verbose:    fileCfg.Verbose,
		StdLog:     fileCfg.StdLog,
		ErrLog:     fileCfg.ErrLog,
	}

	if flag.Lookup("ttl").Changed {
		cfg.TTL = *ttlPtr
	}
	if flag.Lookup("max-entries").Changed {
		cfg.MaxEntries = *maxEntriesPtr
	}
	if flag.Lookup("verbose").Changed {
		cfg.Verbose = *verbosePtr
	}
	if flag.Lookup("stdlog").Changed {
		cfg.StdLog = *stdlogPtr
	}
	if flag.Lookup("errlog").Changed {
		cfg.ErrLog = *errlogPtr
	}
	if flag.Lookup("user").Changed && *userPtr != "" {
		parts := strings.SplitN(*userPtr, ":", 2)
		cfg.Username = parts[0]
		if len(parts) > 1 {
			cfg.Password = parts[1]
		}
	}

	args := flag.Args()

	return cfg, args, nil
}
