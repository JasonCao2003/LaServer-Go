package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var recordingSafeRE = regexp.MustCompile(`[^0-9A-Za-z_.:-]`)

type UGCResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

func (a *Service) recordingDir() string {
	return filepath.Join(a.cfg.InstanceDir, "recordings")
}

func safeRecordingID(id string) string {
	out := recordingSafeRE.ReplaceAllString(id, "")
	if len(out) > 128 {
		out = out[:128]
	}
	return out
}

func newRecordingID(seed string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", seed, time.Now().UnixNano(), len(seed))))
	return hex.EncodeToString(sum[:])[:24]
}

func (a *Service) recordingMetaPath(rid string) string {
	return filepath.Join(a.recordingDir(), safeRecordingID(rid)+".json")
}

func (a *Service) recordingDataPath(rid string) string {
	return filepath.Join(a.recordingDir(), safeRecordingID(rid)+".ndjson")
}

func (a *Service) loadRecordingMeta(rid string) (map[string]any, bool) {
	raw, err := os.ReadFile(a.recordingMetaPath(rid))
	if err != nil {
		return nil, false
	}
	var meta map[string]any
	if json.Unmarshal(raw, &meta) != nil {
		return nil, false
	}
	return meta, true
}

func (a *Service) saveRecordingMeta(rid string, meta map[string]any) bool {
	if safeRecordingID(rid) == "" {
		return false
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		return false
	}
	if err = os.MkdirAll(a.recordingDir(), 0o755); err != nil {
		return false
	}
	path := a.recordingMetaPath(rid)
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, encoded, 0o644); err != nil {
		return false
	}
	return os.Rename(tmp, path) == nil
}

func recordingAuthor(m Object) string {
	uid := text(m, "user")
	claimed := text(m, "user_id")
	placeholder := claimed == ""
	for _, ch := range claimed {
		if ch != '0' && ch != '-' {
			placeholder = false
			break
		}
	}
	if uid != "" && placeholder {
		claimed = uid
	}
	if uid == "" {
		uid = claimed
	}
	if uid == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return uid
}

func recordingName(m Object) string {
	if name := text(m, "name"); name != "" {
		return name
	}
	if title := text(m, "title"); title != "" {
		return title
	}
	return text(m, "message")
}

func recordingData(m Object) map[string]any {
	switch x := m["recording_data"].(type) {
	case map[string]any:
		return x
	case string:
		var out map[string]any
		if json.Unmarshal([]byte(x), &out) == nil {
			return out
		}
	}
	return map[string]any{}
}

func pickRecordingID(m Object, keys ...string) string {
	for _, key := range keys {
		if s := text(m, key); strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
		if obj, ok := m[key].(map[string]any); ok {
			if s := text(obj, "recording_id"); s != "" {
				return s
			}
			if s := text(obj, "id"); s != "" {
				return s
			}
		}
	}
	return ""
}

func (a *Service) recordingLinkBase(r Request) string {
	proto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = "http"
		if r.Secure {
			proto = "https"
		}
	}
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return strings.TrimRight(proto+"://"+host, "/")
}

func firstHeaderValue(v string) string {
	if index := strings.IndexByte(v, ','); index >= 0 {
		v = v[:index]
	}
	return strings.TrimSpace(v)
}

func (a *Service) recordingLink(r Request, rid, author string) string {
	if author == "" {
		author = "_"
	}
	return a.recordingLinkBase(r) + "/ugc/" + author + "/" + safeRecordingID(rid) + ".ndjson"
}

func (a *Service) recordingOKPayload(r Request, rid string, meta map[string]any) Object {
	seek := meta["seek_table"]
	if seek == nil {
		seek = map[string]any{}
	}
	author, _ := meta["user_id"].(string)
	return Object{"status": "OK", "recording_id": meta["recording_id"], "link": a.recordingLink(r, rid, author), "seek_table": seek}
}

func (a *Service) RecordingCreateV1(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	if author := recordingAuthor(m); author != uid {
		return nil, fail(403, "Recording author mismatch")
	}
	a.recordingMu.Lock()
	defer a.recordingMu.Unlock()
	rid := text(m, "recording_id")
	if safeRecordingID(rid) == "" {
		rid = newRecordingID("v1")
	}
	meta := map[string]any{
		"recording_id": rid, "user_id": recordingAuthor(m), "level_id": text(m, "level_id"),
		"name": recordingName(m), "created_at": now(), "recording_data": recordingData(m),
	}
	if !a.saveRecordingMeta(rid, meta) {
		return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
	}
	return Object{"status": "OK", "recording_id": rid}, nil
}

func (a *Service) findRecordingForID(r Request, rid string) (map[string]any, string) {
	meta, found := a.loadRecordingMeta(rid)
	if found {
		return meta, rid
	}
	if rid == "" {
		return nil, ""
	}
	if resource, ok, err := a.store.ResourceIDForFeed(r.Context, feedBareID(rid)); err == nil && ok && resource != rid {
		if meta, found = a.loadRecordingMeta(resource); found {
			return meta, resource
		}
	}
	if resource, ok, err := a.store.ResourceIDForFeed(r.Context, rid); err == nil && ok && resource != rid {
		if meta, found = a.loadRecordingMeta(resource); found {
			return meta, resource
		}
	}
	return nil, rid
}

func (a *Service) RecordingGetV1(r Request, m Object) (Object, error) {
	rid := pickRecordingID(m, "recording_id", "resource_id", "id", "recording")
	meta, _ := a.findRecordingForID(r, rid)
	if meta == nil && rid == "" {
		// latest fallback for v1 clients that only send a user/level.
		files, _ := os.ReadDir(a.recordingDir())
		var best map[string]any
		var bestStamp int64
		userID := text(m, "user")
		if userID == "" {
			userID = text(m, "user_id")
		}
		level := text(m, "level_id")
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}
			candidate, ok := a.loadRecordingMeta(strings.TrimSuffix(file.Name(), ".json"))
			if !ok {
				continue
			}
			author, _ := candidate["user_id"].(string)
			if userID != "" && author != "" && author != userID {
				continue
			}
			storedLevel, _ := candidate["level_id"].(string)
			if level != "" && storedLevel != "" && storedLevel != level {
				continue
			}
			stamp := number(candidate, "created_at", 0)
			if best == nil || stamp > bestStamp {
				best, bestStamp = candidate, stamp
			}
		}
		meta = best
	}
	if meta == nil {
		return Object{"status": "NOT_FOUND", "recording_id": rid}, nil
	}
	return Object{"level_id": meta["level_id"], "recording_data": meta["recording_data"], "recording_id": meta["recording_id"], "status": "OK", "user_id": meta["user_id"]}, nil
}

func (a *Service) RecordingCreateV2(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	if author := recordingAuthor(m); author != uid {
		return nil, fail(403, "Recording author mismatch")
	}
	a.recordingMu.Lock()
	defer a.recordingMu.Unlock()
	rid := text(m, "recording_id")
	if safeRecordingID(rid) == "" {
		rid = newRecordingID("v2")
	}
	author := recordingAuthor(m)
	existing, found := a.loadRecordingMeta(rid)
	if found {
		storedAuthor, _ := existing["user_id"].(string)
		if storedAuthor != "" && storedAuthor != author {
			return nil, &Error{Status: 409, Body: Object{"status": "ERROR", "error": "RECORDING_ID_CONFLICT", "recording_id": rid}}
		}
		if table, ok := existing["seek_table"].(map[string]any); ok && len(table) > 0 {
			return a.recordingOKPayload(r, rid, existing), nil
		}
	}
	data := recordingData(m)
	line, _ := json.Marshal(data)
	if err := os.MkdirAll(a.recordingDir(), 0o755); err != nil {
		return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
	}
	if err := os.WriteFile(a.recordingDataPath(rid), line, 0o644); err != nil {
		return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
	}
	created := now()
	if found {
		if old, ok := existing["created_at"].(float64); ok {
			created = int64(old)
		}
	}
	meta := map[string]any{
		"recording_id": rid, "user_id": author, "level_id": text(m, "level_id"), "pool_type": text(m, "pool_type"),
		"name": recordingName(m), "created_at": created, "updated_at": now(), "recording_data": data,
		"seek_table":  map[string]any{"0": map[string]any{"start": 0, "end": len(line) - 1}},
		"chunk_count": 1, "last_chet": data["chet"], "mode": "v2",
	}
	if meta["last_chet"] == nil {
		meta["last_chet"] = 0
	}
	if !a.saveRecordingMeta(rid, meta) {
		if !found {
			_ = os.Remove(a.recordingDataPath(rid))
		}
		return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
	}
	return a.recordingOKPayload(r, rid, meta), nil
}

func (a *Service) RecordingUpdateV2(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	if author := recordingAuthor(m); author != uid {
		return nil, fail(403, "Recording author mismatch")
	}
	a.recordingMu.Lock()
	defer a.recordingMu.Unlock()
	rid := strings.TrimSpace(text(m, "recording_id"))
	if rid == "" {
		return nil, &Error{Status: 400, Body: Object{"status": "ERROR", "error": "MISSING_RECORDING_ID"}}
	}
	chunk := number(m, "chunk", -1)
	if chunk < 0 {
		return nil, &Error{Status: 400, Body: Object{"status": "ERROR", "error": "INVALID_CHUNK"}}
	}
	meta, found := a.loadRecordingMeta(rid)
	if !found {
		return nil, &Error{Status: 404, Body: Object{"status": "ERROR", "error": "RECORDING_NOT_FOUND", "recording_id": rid}}
	}
	storedAuthor, _ := meta["user_id"].(string)
	if storedAuthor != "" && storedAuthor != uid {
		return nil, &Error{Status: 403, Body: Object{"status": "ERROR", "error": "RECORDING_FORBIDDEN", "recording_id": rid}}
	}
	table := map[string]any{}
	if chunk != 0 {
		if existing, ok := meta["seek_table"].(map[string]any); ok {
			table = existing
		}
	}
	if chunk != 0 {
		if _, ok := table[strconv.FormatInt(chunk, 10)]; ok {
			return a.recordingOKPayload(r, rid, meta), nil
		}
		if int64(len(table)) != chunk {
			return nil, &Error{Status: 409, Body: Object{"status": "ERROR", "error": "OUT_OF_ORDER_CHUNK", "expected": len(table), "received": chunk, "recording_id": rid}}
		}
	}
	data := recordingData(m)
	line, _ := json.Marshal(data)
	path := a.recordingDataPath(rid)
	start := int64(0)
	if chunk == 0 {
		if err := os.WriteFile(path, line, 0o644); err != nil {
			return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
		}
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
		}
		start = info.Size() + 1
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
		}
		_, err = file.Write(append([]byte("\n"), line...))
		file.Close()
		if err != nil {
			return nil, &Error{Status: 500, Body: Object{"status": "ERROR", "error": "STORAGE_ERROR", "recording_id": rid}}
		}
	}
	table[strconv.FormatInt(chunk, 10)] = map[string]any{"start": start, "end": start + int64(len(line)) - 1}
	meta["seek_table"] = table
	meta["chunk_count"] = len(table)
	if data["chet"] != nil {
		meta["last_chet"] = data["chet"]
	} else if meta["last_chet"] == nil {
		meta["last_chet"] = 0
	}
	meta["updated_at"] = now()
	meta["mode"] = "v2"
	if chunk == 0 {
		meta["recording_data"] = data
	}
	_ = a.saveRecordingMeta(rid, meta)
	return a.recordingOKPayload(r, rid, meta), nil
}

func (a *Service) RecordingGetV2(r Request, m Object) (Object, error) {
	rid := pickRecordingID(m, "recording_id", "resource_id", "id", "recording")
	meta, resolved := a.findRecordingForID(r, rid)
	if meta == nil {
		return Object{"status": "NOT_FOUND", "recording_id": rid}, nil
	}
	table, _ := meta["seek_table"].(map[string]any)
	if len(table) == 0 {
		if raw, err := os.ReadFile(a.recordingDataPath(resolved)); err == nil {
			table = map[string]any{}
			offset := int64(0)
			for index, line := range strings.Split(string(raw), "\n") {
				if line == "" {
					offset++
					continue
				}
				length := int64(len([]byte(line)))
				table[strconv.Itoa(index)] = map[string]any{"start": offset, "end": offset + length - 1}
				offset += length + 1
			}
			if len(table) > 0 {
				meta["seek_table"] = table
				_ = a.saveRecordingMeta(resolved, meta)
			}
		}
	}
	recmID := sha1.Sum([]byte(fmt.Sprint(meta["recording_id"])))
	return Object{"MongoID": hex.EncodeToString(recmID[:])[:24], "status": "OK", "recording_id": meta["recording_id"], "seek_table": table, "link": a.recordingLink(r, fmt.Sprint(meta["recording_id"]), fmt.Sprint(meta["user_id"])), "user_id": meta["user_id"], "level_id": meta["level_id"]}, nil
}

func parseRange(header string, total int64) (int64, int64, bool, bool) {
	if header == "" {
		return 0, total - 1, false, true
	}
	if !strings.HasPrefix(strings.ToLower(header), "bytes=") {
		return 0, 0, false, false
	}
	spec := strings.Split(strings.TrimSpace(header[6:]), ",")[0]
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false, false
	}
	a, b := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	var start, end int64
	if a == "" {
		n, err := strconv.ParseInt(b, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false, false
		}
		start = total - n
		if start < 0 {
			start = 0
		}
		end = total - 1
	} else {
		parsed, err := strconv.ParseInt(a, 10, 64)
		if err != nil || parsed < 0 {
			return 0, 0, false, false
		}
		start = parsed
		if b == "" {
			end = total - 1
		} else {
			parsedEnd, err := strconv.ParseInt(b, 10, 64)
			if err != nil {
				return 0, 0, false, false
			}
			end = parsedEnd
		}
	}
	if start > end || start >= total {
		return 0, 0, false, false
	}
	if end >= total {
		end = total - 1
	}
	return start, end, true, true
}

func (a *Service) ServeUGC(subpath, rangeHeader string) UGCResponse {
	for _, part := range strings.Split(subpath, "/") {
		if part == "" || part == "." || part == ".." || strings.Contains(part, `\`) || safeRecordingID(part) != part {
			return UGCResponse{Status: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: []byte("not found")}
		}
	}
	name := subpath
	if index := strings.LastIndexAny(name, `/\`); index >= 0 {
		name = name[index+1:]
	}
	if name == "" {
		return UGCResponse{Status: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: []byte("not found")}
	}
	path := filepath.Join(a.recordingDir(), safeRecordingID(name))
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(a.recordingDir())+string(os.PathSeparator)) {
		return UGCResponse{Status: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: []byte("not found")}
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return UGCResponse{Status: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: []byte("not found")}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return UGCResponse{Status: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: []byte("not found")}
	}
	headers := map[string]string{"Accept-Ranges": "bytes", "Cache-Control": "no-cache", "Content-Type": "text/plain; charset=utf-8"}
	start, end, ranged, ok := parseRange(rangeHeader, int64(len(data)))
	if !ok {
		headers["Content-Range"] = fmt.Sprintf("bytes */%d", len(data))
		return UGCResponse{Status: 416, Headers: headers}
	}
	if !ranged {
		return UGCResponse{Status: 200, Headers: headers, Body: data}
	}
	headers["Content-Range"] = fmt.Sprintf("bytes %d-%d/%d", start, end, len(data))
	return UGCResponse{Status: 206, Headers: headers, Body: data[start : end+1]}
}

var _ = fmt.Sprintf
