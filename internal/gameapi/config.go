package gameapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func LoadConfig(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return cfg, errors.New("config must contain one JSON object")
	}
	if _, _, err = net.SplitHostPort(cfg.Listen); err != nil {
		return cfg, fmt.Errorf("invalid listen: %w", err)
	}
	if cfg.Database == "" {
		return cfg, errors.New("database is required")
	}
	if (cfg.TLSCert == "") != (cfg.TLSKey == "") {
		return cfg, errors.New("tls_cert and tls_key must be configured together")
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return cfg, err
	}
	resolve := func(s string) string {
		if s != "" && !filepath.IsAbs(s) {
			return filepath.Join(base, s)
		}
		return s
	}
	cfg.Database = resolve(cfg.Database)
	cfg.TLSCert = resolve(cfg.TLSCert)
	cfg.TLSKey = resolve(cfg.TLSKey)
	for k, v := range cfg.DefaultCurrency {
		if !currencyField(k) || v < 0 {
			return cfg, fmt.Errorf("invalid default currency %q", k)
		}
	}
	if cfg.CurrentSeason < 0 {
		return cfg, errors.New("current_season cannot be negative")
	}
	return cfg, nil
}
