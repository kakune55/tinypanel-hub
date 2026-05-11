package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Storage.DataFile != "data/tinypanel.json" {
		t.Fatalf("data file = %q", cfg.Storage.DataFile)
	}
	if cfg.Storage.TelemetryFile != "data/telemetry.jsonl" {
		t.Fatalf("telemetry file = %q", cfg.Storage.TelemetryFile)
	}
}

func TestPathFromEnvUsesEtcConfigByDefault(t *testing.T) {
	t.Setenv("TINYPANEL_CONFIG", "")

	if got := PathFromEnv(); got != "etc/config.json" {
		t.Fatalf("default config path = %q, want etc/config.json", got)
	}
}

func TestLoadJSONConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(`{
		"server": {
			"addr": ":9090",
			"api_token": "secret"
		},
		"storage": {
			"data_file": "data/dev.json",
			"telemetry_file": "data/dev-telemetry.jsonl"
		},
		"weather": {
			"provider": "qweather",
			"api_host": "abcxyz.qweatherapi.com",
			"api_key": "key",
			"location": "101020100",
			"hours": "72h",
			"days": "7d"
		}
	}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Addr != ":9090" || cfg.Server.APIToken != "secret" || cfg.Storage.DataFile != "data/dev.json" || cfg.Storage.TelemetryFile != "data/dev-telemetry.jsonl" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Weather.Provider != "qweather" || cfg.Weather.APIHost != "abcxyz.qweatherapi.com" || cfg.Weather.APIKey != "key" || cfg.Weather.Location != "101020100" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Weather.Hours != "72h" || cfg.Weather.Days != "7d" {
		t.Fatalf("unexpected forecast ranges: %+v", cfg.Weather)
	}
}

func TestLoadPartialJSONConfigFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"server":{"api_token":"secret"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("addr = %q, want default", cfg.Server.Addr)
	}
	if cfg.Storage.DataFile != "data/tinypanel.json" {
		t.Fatalf("data file = %q, want default", cfg.Storage.DataFile)
	}
	if cfg.Storage.TelemetryFile != "data/telemetry.jsonl" {
		t.Fatalf("telemetry file = %q, want default", cfg.Storage.TelemetryFile)
	}
	if cfg.Weather.Provider != "manual" || cfg.Weather.Hours != "24h" || cfg.Weather.Days != "3d" || cfg.Weather.CacheTTL != "10m" || cfg.Weather.Timeout != "5s" {
		t.Fatalf("unexpected weather defaults: %+v", cfg.Weather)
	}
}
