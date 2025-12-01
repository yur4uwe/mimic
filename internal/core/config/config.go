package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Mountpoint string `toml:"mpoint"`

	URL      string `toml:"url"`
	Username string `toml:"username"`
	Password string `toml:"password"`

	TTL        time.Duration `toml:"ttl"`
	MaxEntries int           `toml:"max-entries"`

	Verbose bool   `toml:"verbose"`
	StdLog  string `toml:"std"`
	ErrLog  string `toml:"err"`
}

const defaultConfig = `# server
username = "user"
password = "pass"

# cache
ttl = "1s" # important to be in quotes!
max-entries = 100

# logger
verbose = false
err = "stderr"
std = "stdout"
`

// On Linux/macOS uses XDG_CONFIG_HOME or ~/.config; on Windows uses %APPDATA%.
func userConfigPath(appName, filename string) (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
		if base == "" {
			home := os.Getenv("USERPROFILE")
			if home == "" {
				return "", fmt.Errorf("cannot determine APPDATA or USERPROFILE")
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
	} else {
		base = os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home := os.Getenv("HOME")
			if home == "" {
				return "", fmt.Errorf("cannot determine HOME")
			}
			base = filepath.Join(home, ".config")
		}
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, filename), nil
}

func writeDefaultConfig(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ParseConfig(path string) (*Config, error) {
	var cfg Config

	if path == "" {
		// No config path provided; try per-user config.
		defCfgPath, perr := userConfigPath("mimic", "config.toml")
		if perr != nil {
			return nil, perr
		}

		// If the per-user config doesn't exist, try creating a default one.
		if _, statErr := os.Stat(defCfgPath); os.IsNotExist(statErr) {
			fmt.Println("Missing per user config, trying to create a new one at", defCfgPath)
			if werr := writeDefaultConfig(defCfgPath, defaultConfig); werr != nil {
				return nil, werr
			}
		}

		// Use the per-user config path (either existing or newly created).
		path = defCfgPath
	}

	if path == "" {
		return nil, fmt.Errorf("no config path provided and no per-user config available")
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <mountpoint> <server>\n", os.Args[0])
	flag.PrintDefaults()
}

func ParseCommandLineArgs() (*Config, error) {
	var (
		configFPtr    = flag.StringP("config", "c", "", "path to config file")
		userPtr       = flag.StringP("user", "u", "", "username:password (shorthand)")
		ttlPtr        = flag.DurationP("ttl", "t", time.Minute, "cache TTL")
		maxEntriesPtr = flag.IntP("max-entries", "m", 1000, "cache max entries")
		verbosePtr    = flag.BoolP("verbose", "v", false, "enable verbose logging")
		stdlogPtr     = flag.StringP("stdlog", "s", "", "path to standard log file")
		errlogPtr     = flag.StringP("errlog", "e", "", "path to error log file")
		wherePtr      = flag.Bool("where-config", false, "print the path to the config file and exit")
	)

	flag.Usage = usage
	flag.Parse()

	cfg, err := ParseConfig(*configFPtr)
	if err != nil {
		return nil, err
	}

	if *wherePtr {
		p, perr := userConfigPath("mimic", "config.toml")
		if perr != nil {
			return nil, perr
		}
		fmt.Println(p)
		os.Exit(0)
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

	if len(args) == 2 && cfg.Mountpoint == "" {
		cfg.Mountpoint = args[0]
	}

	if len(args) == 2 && cfg.URL == "" {
		cfg.URL = args[1]
	}

	return cfg, nil
}
