package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func task3CallWithUA(t *testing.T, api *API, path, ua string, body map[string]any, want int) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", path, bytes.NewReader(raw))
	req.Header.Set("User-Agent", ua)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", ua, path, w.Code, want, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTask3EventScheduleKeepsLegacyTimeWithinSigned32Bit(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	for name, value := range map[string]string{
		"event_schedule_low.json":  `{"event_schedule":{"base_time":1587366000,"valid_until":2147483648,"events":[{"name":"low_only","when":[9999999999],"duration":"1"}]}}`,
		"event_schedule_high.json": `{"event_schedule":{"base_time":1587366000,"valid_until":2147483648,"events":[{"name":"high_only","when":[9999999999],"duration":"1"}]}}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, ua := range []string{"Sky-Live-ios/0.1.8.100", "Sky-Live-ios/0.3.11.100", "Sky-Live-ios/0.13.4.170388"} {
		out := task3CallWithUA(t, api, "/account/get_event_schedule", ua, map[string]any{}, 200)
		schedule := out["event_schedule"].(map[string]any)
		if until := schedule["valid_until"].(float64); until > 2147483647 {
			t.Fatalf("%s: valid_until overflows old client: %.0f", ua, until)
		}
		entry := schedule["events"].([]any)[0].(map[string]any)
		if entry["name"] != "low_only" {
			t.Fatalf("%s: selected wrong schedule: %v", ua, entry)
		}
		if offset := entry["when"].([]any)[0].(float64); offset > 2147483647-1587366000 {
			t.Fatalf("%s: event offset overflows old client: %.0f", ua, offset)
		}
	}
	out := task3CallWithUA(t, api, "/account/get_event_schedule", "Sky-Live-ios/0.26.0.100", map[string]any{}, 200)
	entry := out["event_schedule"].(map[string]any)["events"].([]any)[0].(map[string]any)
	if entry["name"] != "high_only" {
		t.Fatalf("modern client selected wrong schedule: %v", entry)
	}
}

func TestTask3EventScheduleUsesConfiguredBaseTime(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	fixture := `{"event_schedule":{"base_time":1600000000,"valid_until":2000000000,"events":[{"name":"custom","when":[10],"duration":"1"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "event_schedule_low.json"), []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	out := task3CallWithUA(t, api, "/account/get_event_schedule", "Sky-Live-ios/0.3.11.100", map[string]any{}, 200)
	schedule := out["event_schedule"].(map[string]any)
	if schedule["base_time"] != float64(1600000000) {
		t.Fatalf("ignored configured base_time: %v", schedule["base_time"])
	}
}

func TestTask3EventScheduleFallsBackWhenConfigIsMissingOrCorrupt(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	for _, fixture := range []string{"", "{", "{}", `{"event_schedule":{"base_time":1600000000,"events":[]}}garbage`} {
		if fixture != "" {
			if err := os.WriteFile(filepath.Join(dir, "event_schedule_low.json"), []byte(fixture), 0600); err != nil {
				t.Fatal(err)
			}
		}
		out := task3CallWithUA(t, api, "/account/get_event_schedule", "Sky-Live-ios/0.1.8.100", map[string]any{}, 200)
		schedule, ok := out["event_schedule"].(map[string]any)
		if !ok || schedule["base_time"] != float64(1587366000) {
			t.Fatalf("fixture=%q: missing compatible fallback schedule: %v", fixture, out)
		}
	}
}

func TestTask3EventScheduleIgnoresInvalidPreferredSource(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "event_schedule.json"), []byte(`{"event_schedule":{"base_time":1600000000,"events":[{"name":"fallback","when":[1],"duration":"1"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`{"event_schedule":{"base_time":1600000000,"events":"bad"}}`,
		`{"event_schedule":{"base_time":"bad","events":[]}}`,
		`{"event_schedule":{"base_time":1600000000,"events":[null]}}`,
		`{"event_schedule":{"events":[{}]}}`,
		`{"event_schedule":{"events":[{"name":"bad","when":"bad","duration":"1"}]}}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, "event_schedule_low.json"), []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		out := task3CallWithUA(t, api, "/account/get_event_schedule", "Sky-Live-ios/0.3.11.100", map[string]any{}, 200)
		events := out["event_schedule"].(map[string]any)["events"].([]any)
		if len(events) != 1 || events[0].(map[string]any)["name"] != "fallback" {
			t.Fatalf("invalid preferred source %q hid valid fallback: %v", bad, out)
		}
	}
}
