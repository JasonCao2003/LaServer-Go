package service

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const disabledEventOffset = int64(9999999999)

var seasonEventName = regexp.MustCompile(`^season_0*([0-9]+)(?:_.*)?$`)

func asObject(value any) Object { out, _ := value.(map[string]any); return out }
func asObjects(value any) []Object {
	entries, _ := value.([]any)
	out := make([]Object, 0, len(entries))
	for _, entry := range entries {
		if obj := asObject(entry); obj != nil {
			out = append(out, obj)
		}
	}
	return out
}
func boolSetting(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return fallback
}
func tcandleGroup(name string) string {
	if strings.HasSuffix(name, "_tcandles1") || strings.HasSuffix(name, "_tcandles2") {
		return "candles"
	}
	if strings.HasSuffix(name, "_tcandle_master") {
		return "tcandle_master"
	}
	return ""
}

func (a *Service) scheduleSource(ua string) Object {
	if a.cfg.GameConfigDir == "" {
		return nil
	}
	major, minor, _, known := clientVersion(ua)
	high := known && (major > 0 || minor >= 26)
	preferred := "event_schedule_low.json"
	if high {
		preferred = "event_schedule_high.json"
	}
	for _, name := range []string{preferred, "event_schedule.json"} {
		raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, name))
		if err != nil {
			continue
		}
		var value Object
		if json.Unmarshal(raw, &value) == nil && asObject(value["event_schedule"]) != nil {
			return value
		}
	}
	return nil
}

func scheduledNames(events []Object, group string) []string {
	out := []string{}
	for _, event := range events {
		if tcandleGroup(text(event, "name")) == group {
			out = append(out, text(event, "name"))
		}
	}
	sort.Strings(out)
	return out
}
func selectedDaily(names []string, count int, day, group string) map[string]bool {
	out := map[string]bool{}
	if count > len(names) {
		count = len(names)
	}
	if count < 0 {
		count = 0
	}
	type scored struct {
		name  string
		score uint64
	}
	scores := make([]scored, 0, len(names))
	for _, name := range names {
		h := fnv.New64a()
		_, _ = h.Write([]byte(day + ":" + group + ":" + name))
		scores = append(scores, scored{name, h.Sum64()})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score < scores[j].score })
	for _, entry := range scores[:count] {
		out[entry.name] = true
	}
	return out
}

func applyCandleSchedule(events []Object, control Object, base, nowUnix, midnight int64) []Object {
	tc := asObject(control["tcandles"])
	if tc == nil {
		return events
	}
	seen := map[string]bool{}
	for _, event := range events {
		seen[text(event, "name")] = true
	}
	for _, realm := range []string{"prairie", "rain", "sunset", "dusk", "night"} {
		for _, kind := range []string{"tcandles1", "tcandles2", "tcandle_master"} {
			name := "event_" + realm + "_" + kind
			if !seen[name] {
				events = append(events, Object{"name": name, "when": []int64{disabledEventOffset}, "duration": "1440"})
				seen[name] = true
			}
		}
	}
	groups := asObject(tc["groups"])
	types := asObject(tc["types"])
	for _, group := range []string{"candles", "tcandle_master"} {
		settings := asObject(groups[group])
		if settings == nil && types != nil {
			kind := group
			if group == "candles" {
				kind = "tcandles1"
			}
			settings = asObject(types[kind])
		}
		if settings == nil {
			settings = tc
		}
		all := boolSetting(settings["all_enable"], false)
		auto := boolSetting(settings["auto"], false)
		if groups == nil && types != nil && group == "candles" {
			second := asObject(types["tcandles2"])
			all = all || boolSetting(second["all_enable"], false)
			auto = auto || boolSetting(second["auto"], false)
			if number(settings, "auto_count", 0) == 0 && second != nil {
				settings["auto_count"] = second["auto_count"]
			}
			if number(settings, "duration_seconds", 0) == 0 && second != nil {
				settings["duration_seconds"] = second["duration_seconds"]
			}
		}
		if groups == nil && types == nil && group == "candles" {
			all = boolSetting(tc["all_enable_candles1"], false) || boolSetting(tc["all_enable_candles2"], false)
			auto = text(tc, "mode") == "auto"
		}
		enabled := map[string]bool{}
		names := scheduledNames(events, group)
		if boolSetting(tc["enabled"], true) {
			if all {
				for _, name := range names {
					enabled[name] = true
				}
			} else if auto {
				enabled = selectedDaily(names, int(number(settings, "auto_count", 2)), time.Unix(midnight, 0).Format("2006-01-02"), group)
			}
		}
		duration := number(settings, "duration_seconds", 86400) / 60
		if duration < 1 {
			duration = 1
		}
		for _, event := range events {
			if tcandleGroup(text(event, "name")) != group {
				continue
			}
			if enabled[text(event, "name")] {
				when := nowUnix - base - 60
				minutes := duration
				if auto && !all {
					when = midnight - base
					minutes = 1440
				}
				event["when"] = []int64{when}
				event["duration"] = strconv.FormatInt(minutes, 10)
			} else {
				event["when"] = []int64{disabledEventOffset}
			}
		}
	}
	return events
}

func applySeasonSchedule(events []Object, control Object, base, nowUnix, midnight int64) []Object {
	season := asObject(control["season"])
	current := number(season, "current_season", 0)
	start := number(season, "start_time", 0)
	end := start + number(season, "duration_days", 70)*86400
	force := boolSetting(season["force_end"], false)
	when := []int64{midnight - base - 7*86400}
	duration := "99999999"
	if start > 0 {
		when = []int64{start - base}
		if end > start {
			duration = strconv.FormatInt((end-start)/60, 10)
		}
	}
	if force && start <= 0 {
		when = []int64{nowUnix - base - 8*86400}
		duration = "10080"
	}
	if force && start > 0 {
		minutes := (nowUnix - start) / 60
		if minutes < 1 {
			minutes = 1
		}
		duration = strconv.FormatInt(minutes, 10)
	}
	foundMain, foundMarker := false, false
	for _, event := range events {
		name := text(event, "name")
		if name == "global_daily_reset_time" {
			event["when"] = []int64{midnight - base - 86400, midnight - base, midnight - base + 86400}
			event["duration"] = "1440"
			continue
		}
		matches := seasonEventName.FindStringSubmatch(name)
		if len(matches) == 0 {
			continue
		}
		num, _ := strconv.ParseInt(matches[1], 10, 64)
		marker := strings.Contains(name, "_ever_started") || strings.Contains(name, "_ever_completed")
		if marker {
			if num == current && current > 0 {
				event["when"] = []int64{0}
				event["duration"] = "99999999"
				if strings.Contains(name, "_ever_started") {
					foundMarker = true
				}
			} else {
				event["when"] = []int64{disabledEventOffset}
				event["duration"] = "0"
			}
			continue
		}
		if num == current && current > 0 && (name == fmt.Sprintf("season_%02d", current) || name == fmt.Sprintf("season_%d", current)) {
			event["when"] = when
			event["duration"] = duration
			foundMain = true
		} else {
			event["when"] = []int64{disabledEventOffset}
			event["duration"] = "0"
		}
	}
	if current > 0 && !foundMain {
		events = append(events, Object{"name": fmt.Sprintf("season_%02d", current), "when": when, "duration": duration})
	}
	if current > 0 && !foundMarker {
		events = append(events, Object{"name": fmt.Sprintf("season_%02d_ever_started", current), "when": []int64{0}, "duration": "99999999"})
	}
	return events
}

func (a *Service) EventSchedule(r Request, _ Object) (Object, error) {
	data := a.scheduleSource(r.UserAgent)
	schedule := asObject(data["event_schedule"])
	if schedule == nil {
		schedule = Object{"base_time": int64(1587366000), "valid_until": int64(2147483648), "events": []any{}}
	}
	base := number(schedule, "base_time", 1587366000)
	events := asObjects(schedule["events"])
	control, _ := a.legacyObject("game_control.json")
	nowTime := time.Now()
	nowUnix := nowTime.Unix()
	year, month, day := nowTime.Date()
	midnight := time.Date(year, month, day, 0, 0, 0, 0, nowTime.Location()).Unix()
	events = applyCandleSchedule(events, control, base, nowUnix, midnight)
	events = applySeasonSchedule(events, control, base, nowUnix, midnight)
	if settings := asObject(control["events"]); settings != nil && !boolSetting(settings["enabled"], true) {
		for _, event := range events {
			name := strings.ToLower(text(event, "name"))
			if name == "global_daily_reset_time" || seasonEventName.MatchString(name) || tcandleGroup(name) != "" {
				continue
			}
			for _, prefix := range []string{"spiritshopitem", "visiting_vendor", "visitingvendor", "skyfest", "aurora", "ap18_ritual", "feast", "te_ep", "fireworks", "nature_geyser", "day_of_", "days_of_", "day_", "days_", "global_"} {
				if strings.HasPrefix(name, prefix) {
					event["when"] = []int64{disabledEventOffset}
					event["duration"] = "0"
					break
				}
			}
		}
	}
	if a.cfg.GameConfigDir != "" {
		if _, err := os.Stat(filepath.Join(a.cfg.GameConfigDir, "no_season")); err == nil {
			kept := events[:0]
			for _, event := range events {
				if !strings.Contains(strings.ToLower(text(event, "name")), "season") {
					kept = append(kept, event)
				}
			}
			events = kept
			if len(events) == 0 {
				return Object{}, nil
			}
		}
	}
	out := Object{"base_time": base, "events": events}
	major, minor, _, known := clientVersion(r.UserAgent)
	if known && (major > 0 || minor >= 26) {
		out["server_time"] = nowUnix
		out["valid_until"] = nowUnix + 1800
	} else {
		out["valid_until"] = number(schedule, "valid_until", 2147483648)
	}
	return Object{"event_schedule": out}, nil
}
