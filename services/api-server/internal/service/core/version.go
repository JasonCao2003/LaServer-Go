// 客户端版本解析

package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`Sky[^/]*/([0-9]+)\.([0-9]+)\.([0-9]+)`)
var rawClientVersion = regexp.MustCompile(`/(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)
var clientVersionRE = regexp.MustCompile(`(?i)Sky[^/]*/([0-9]+)\.([0-9]+)\.([0-9]+)`)

// 判断是否为旧版客户端
func LegacyVersion(ua string) bool {
	m := versionPattern.FindStringSubmatch(ua)
	if len(m) == 0 {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major == 0 && minor < 5
}

// 解析客户端版本号
func ClientVersion(ua string) (int, int, int, bool) {
	m := clientVersionRE.FindStringSubmatch(ua)
	if len(m) != 4 {
		return 0, 0, 0, false
	}
	a, e1 := strconv.Atoi(m[1])
	b, e2 := strconv.Atoi(m[2])
	c, e3 := strconv.Atoi(m[3])
	return a, b, c, e1 == nil && e2 == nil && e3 == nil
}

// 判断是否为旧版客户端（含配置覆盖）
func (a *Base) LegacyUA(r Request) bool {
	match := rawClientVersion.FindStringSubmatch(r.UserAgent)
	if len(match) == 0 {
		return false
	}
	version := strings.Join(match[1:4], ".")
	if a.Cfg.GameConfigDir != "" {
		raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "client_version_overrides.json"))
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
		return LegacyVersion(r.UserAgent)
	}
	major, e1 := strconv.Atoi(parts[0])
	minor, e2 := strconv.Atoi(parts[1])
	_, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return LegacyVersion(r.UserAgent)
	}
	return major == 0 && minor >= 0 && minor < 5
}
