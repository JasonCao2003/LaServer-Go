package service

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var rawClientVersion = regexp.MustCompile(`/(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)

func (a *Service) legacyUA(r Request) bool {
	match := rawClientVersion.FindStringSubmatch(r.UserAgent)
	if len(match) == 0 {
		return false
	}
	version := strings.Join(match[1:4], ".")
	if a.cfg.GameConfigDir != "" {
		raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "client_version_overrides.json"))
		if err == nil {
			var groups map[string]json.RawMessage
			if json.Unmarshal(raw, &groups) == nil {
				full := version
				if match[4] != "" {
					full += "." + match[4]
				}
				found := false
				for _, lookup := range [][2]string{{"by_full", full}, {"by_build", match[4]}, {"by_version", version}} {
					var values map[string]string
					if json.Unmarshal(groups[lookup[0]], &values) == nil {
						if replacement, ok := values[lookup[1]]; ok {
							version = replacement
							found = true
							break
						}
					}
				}
				if !found {
					// Preserve the old JSON object's insertion order for overlapping UA rules.
					decoder := json.NewDecoder(bytes.NewReader(groups["by_ua_contains"]))
					if token, err := decoder.Token(); err == nil && token == json.Delim('{') {
						for decoder.More() {
							key, err := decoder.Token()
							if err != nil {
								break
							}
							var value string
							if decoder.Decode(&value) != nil {
								break
							}
							part, ok := key.(string)
							if ok && strings.Contains(r.UserAgent, part) {
								version = value
								break
							}
						}
					}
				}
			}
		}
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return legacyVersion(r.UserAgent)
	}
	major, e1 := strconv.Atoi(parts[0])
	minor, e2 := strconv.Atoi(parts[1])
	_, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return legacyVersion(r.UserAgent)
	}
	return major == 0 && minor >= 0 && minor < 5
}
