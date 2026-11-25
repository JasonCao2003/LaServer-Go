package account

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var motdWrite sync.Mutex

func (a *Service) localMotd() (object, error) {
	if a.Cfg.InstanceDir == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(filepath.Join(a.Cfg.InstanceDir, "motd.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data object
	err = json.Unmarshal(raw, &data)
	return data, err
}

func (a *Service) AdminMotd(r Request, m object) (object, error) {
	token := a.Cfg.AdminToken
	ip := net.ParseIP(r.RemoteIP)
	if (token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(r.Header.Get("X-Admin-Token"))) != 1) || (token == "" && !ip.IsLoopback()) {
		return nil, &Error{Status: 403, Body: object{"ok": false}}
	}
	if text(m, "action") == "read" {
		data, err := a.Motd(r, nil)
		return object{"ok": err == nil, "announcement": data}, err
	}
	title, body := strings.TrimSpace(text(m, "title")), text(m, "body")
	version, err := strconv.ParseInt(text(m, "version"), 10, 64)
	if text(m, "action") != "save" || title == "" || len(title) > 256 || len(body) > 16384 || err != nil || version < 1 {
		return nil, &Error{Status: 400, Body: object{"ok": false}}
	}
	buttons := []object{}
	for _, line := range strings.Split(text(m, "buttons"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		link, img, _ := strings.Cut(strings.TrimSpace(line), "|")
		u, err := url.Parse(strings.TrimSpace(link))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || len(line) > 2048 || len(buttons) >= 20 {
			return nil, &Error{Status: 400, Body: object{"ok": false}}
		}
		buttons = append(buttons, object{"url": u.String(), "img": strings.TrimSpace(img)})
	}
	motdWrite.Lock()
	defer motdWrite.Unlock()
	current, err := a.Motd(r, nil)
	if err != nil {
		return nil, err
	}
	if version <= number(current, "motd_version", 0) {
		return nil, &Error{Status: 409, Body: object{"ok": false}}
	}
	current["motd_title"], current["motd"], current["motd_buttons"] = title, body, buttons
	current["motd_version"], current["motd_timestamp"], current["motd_timestamp_mode"] = version, now(), "fixed"
	if a.Cfg.InstanceDir == "" {
		return nil, errors.New("instance directory missing")
	}
	if err := os.MkdirAll(a.Cfg.InstanceDir, 0700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(a.Cfg.InstanceDir, "motd-*.tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(current)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := os.Rename(f.Name(), filepath.Join(a.Cfg.InstanceDir, "motd.json")); err != nil {
		return nil, err
	}
	return object{"ok": true}, nil
}
