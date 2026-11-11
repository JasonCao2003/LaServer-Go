package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func Load(path string) (Config, error) {
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
	cfg.GameConfigDir = resolve(cfg.GameConfigDir)
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

type Config struct {
	Listen                 string           `json:"listen"`
	Database               string           `json:"database"`
	TLSCert                string           `json:"tls_cert"`
	TLSKey                 string           `json:"tls_key"`
	AutoAuthorize          bool             `json:"auto_authorize"`
	RecreateMissingAccount bool             `json:"recreate_missing_account"`
	RoomHost               string           `json:"room_host"`
	RoomPort               int              `json:"room_port"`
	CurrentSeason          int              `json:"current_season"`
	DefaultCurrency        map[string]int64 `json:"default_currency"`
	GameConfigDir          string           `json:"game_config_dir"`
	ResourceAuthToken      string           `json:"resource_auth_token"`
	StarwatchJWTSecret     string           `json:"starwatch_jwt_secret"`
	StarwatchBrokerURL     string           `json:"starwatch_broker_url"`
}

var currencyFields = []string{"candles", "heart", "heart_wax", "prestige", "prestige_vip", "prestige_wax", "season_candle", "season_wax", "season_wax_14", "season_heart", "season_pass_token", "vip", "wax"}

func currencyField(k string) bool {
	for _, f := range currencyFields {
		if f == k {
			return true
		}
	}
	return false
}

func ValidateRuntime(cfg Config) error {
	ip := net.ParseIP(cfg.RoomHost)
	if ip == nil || ip.To4() == nil || cfg.RoomPort < 1 || cfg.RoomPort > 65535 {
		return errors.New("room_host must be IPv4 and room_port must be 1..65535")
	}
	return nil
}
