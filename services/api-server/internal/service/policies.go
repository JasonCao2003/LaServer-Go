package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// legacyObject reads only named configuration files. It never writes legacy state.
func (a *Service) legacyObject(name string) (object, error) {
	if name != filepath.Base(name) {
		return nil, errors.New("invalid configuration name")
	}
	if a.cfg.GameConfigDir == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fail(503, "Game configuration unavailable")
	}
	var out object
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&out) != nil || out == nil {
		return nil, fail(503, "Invalid game configuration")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, fail(503, "Invalid game configuration")
	}
	return out, nil
}

func (a *Service) policy(name string, stamp int64, fallback bool) (bool, error) {
	data, err := a.legacyObject(name)
	if err != nil {
		return false, nil
	}
	if data == nil {
		return fallback, nil
	}
	enabled, ok := data["enabled"].(bool)
	if !ok {
		return false, nil
	}
	since := int64(0)
	if v, ok := data["since"]; ok && v != nil {
		var raw string
		switch x := v.(type) {
		case json.Number:
			raw = x.String()
		case string:
			raw = x
		default:
			return false, nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return false, nil
		}
		since = n
	}
	return enabled && stamp >= since, nil
}
