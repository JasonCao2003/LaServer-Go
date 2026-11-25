package panel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Listen             string
	APIURL, RoomURL    *url.URL
	UpstreamTimeout    time.Duration
	password, apiToken string
	updateDir          string
}

func LoadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open panel config: %w", err)
	}
	defer file.Close()
	input := struct {
		Listen    string `json:"listen"`
		APIURL    string `json:"api_url"`
		RoomURL   string `json:"room_url"`
		TimeoutMS int    `json:"upstream_timeout_ms"`
	}{Listen: "127.0.0.1:18080", TimeoutMS: 2000}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&input); err != nil {
		return Config{}, errors.New("invalid panel config JSON")
	}
	if err = decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return Config{}, errors.New("unexpected data after panel config")
	}
	addr, err := listenAddress(input.Listen)
	if err != nil {
		return Config{}, err
	}
	if input.TimeoutMS < 1 || input.TimeoutMS > 30000 {
		return Config{}, errors.New("upstream_timeout_ms must be between 1 and 30000")
	}
	api, err := parseOrigin(input.APIURL)
	if err != nil {
		return Config{}, fmt.Errorf("api_url: %w", err)
	}
	room, err := parseOrigin(input.RoomURL)
	if err != nil {
		return Config{}, fmt.Errorf("room_url: %w", err)
	}
	password, token := os.Getenv("LASERVER_PANEL_PASSWORD"), os.Getenv("LASERVER_PANEL_API_TOKEN")
	if strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return Config{}, errors.New("LASERVER_PANEL_API_TOKEN must be a nonempty single line")
	}
	return Config{updateDir: filepath.Join(filepath.Dir(path), "update-inbox"), Listen: addr.String(), APIURL: api, RoomURL: room, UpstreamTimeout: time.Duration(input.TimeoutMS) * time.Millisecond, password: password, apiToken: token}, nil
}

func listenAddress(value string) (netip.AddrPort, error) {
	addr, err := netip.ParseAddrPort(value)
	if err != nil || !addr.Addr().IsLoopback() || addr.Addr().Zone() != "" || addr.Port() == 0 {
		return netip.AddrPort{}, errors.New("listen must be a loopback IP address with a nonzero port")
	}
	return addr, nil
}

func parseOrigin(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Opaque != "" || (u.Path != "" && u.EscapedPath() != "/") || u.ForceQuery || u.RawQuery != "" || strings.Contains(value, "#") {
		return nil, errors.New("expected an origin without credentials, path, query or fragment")
	}
	ip, ipErr := netip.ParseAddr(u.Hostname())
	if u.Scheme != "https" && (u.Scheme != "http" || ipErr != nil || !ip.IsLoopback()) {
		return nil, errors.New("expected HTTPS or HTTP on a loopback IP")
	}
	u.Path = ""
	return u, nil
}
