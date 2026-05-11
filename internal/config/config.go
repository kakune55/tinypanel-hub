package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
)

const DefaultPath = "config.json"

type Config struct {
	Server  Server  `json:"server"`
	Storage Storage `json:"storage"`
	Weather Weather `json:"weather"`
}

type Server struct {
	Addr     string `json:"addr"`
	APIToken string `json:"api_token"`
}

type Storage struct {
	DataFile string `json:"data_file"`
}

type Weather struct {
	Provider    string `json:"provider"`
	APIHost     string `json:"api_host"`
	APIKey      string `json:"api_key"`
	BearerToken string `json:"bearer_token"`
	Location    string `json:"location"`
	Lang        string `json:"lang"`
	Unit        string `json:"unit"`
	Hours       string `json:"hours"`
	Days        string `json:"days"`
	CacheTTL    string `json:"cache_ttl"`
	Timeout     string `json:"timeout"`
}

func Default() Config {
	return Config{
		Server: Server{
			Addr: ":8080",
		},
		Storage: Storage{
			DataFile: "data/tinypanel.json",
		},
		Weather: Weather{
			Provider: "manual",
			Lang:     "zh",
			Unit:     "m",
			Hours:    "24h",
			Days:     "3d",
			CacheTTL: "10m",
			Timeout:  "5s",
		},
	}
}

func PathFromEnv() string {
	if v := os.Getenv("TINYPANEL_CONFIG"); v != "" {
		return v
	}
	return DefaultPath
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = DefaultPath
	}

	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return cfg, nil
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return cfg, errors.New("config file must contain a single JSON object")
	}
	cfg.fillDefaults()
	return cfg, nil
}

func (c *Config) fillDefaults() {
	def := Default()
	if c.Server.Addr == "" {
		c.Server.Addr = def.Server.Addr
	}
	if c.Storage.DataFile == "" {
		c.Storage.DataFile = def.Storage.DataFile
	}
	if c.Weather.Provider == "" {
		c.Weather.Provider = def.Weather.Provider
	}
	if c.Weather.Lang == "" {
		c.Weather.Lang = def.Weather.Lang
	}
	if c.Weather.Unit == "" {
		c.Weather.Unit = def.Weather.Unit
	}
	if c.Weather.Hours == "" {
		c.Weather.Hours = def.Weather.Hours
	}
	if c.Weather.Days == "" {
		c.Weather.Days = def.Weather.Days
	}
	if c.Weather.CacheTTL == "" {
		c.Weather.CacheTTL = def.Weather.CacheTTL
	}
	if c.Weather.Timeout == "" {
		c.Weather.Timeout = def.Weather.Timeout
	}
}
