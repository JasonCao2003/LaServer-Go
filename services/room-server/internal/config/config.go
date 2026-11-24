// 房间服务配置加载与校验

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"os"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

// Config contains the room service's local listeners and session limits.
type Config struct {
	UDPListen        string `json:"udp_listen"`
	ManagementListen string `json:"management_listen"`
	MaxPeers         int    `json:"max_peers"`
	MaxPlayers       int    `json:"max_players"`
	JoinTimeoutMS    int64  `json:"join_timeout_ms"`
	MaxFrameBytes    int    `json:"max_frame_bytes"`
}

// 加载并解析服务配置
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open room config: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode room config: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		if err != nil {
			return Config{}, fmt.Errorf("invalid data after room config: %w", err)
		}
		return Config{}, errors.New("unexpected data after room config object")
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate room config: %w", err)
	}
	return cfg, nil
}

// 校验服务配置
func (cfg Config) validate() error {
	udp, err := netip.ParseAddrPort(cfg.UDPListen)
	if err != nil || udp.Port() == 0 {
		return errors.New("udp_listen must be an IP address and a nonzero port")
	}
	management, err := netip.ParseAddrPort(cfg.ManagementListen)
	if err != nil || management.Port() == 0 || !management.Addr().Unmap().IsLoopback() {
		return errors.New("management_listen must be a loopback IP address and a nonzero port")
	}
	if cfg.MaxPeers < 1 {
		return errors.New("max_peers must be positive")
	}
	if cfg.MaxPlayers < 1 || cfg.MaxPlayers > cfg.MaxPeers {
		return errors.New("max_players must be between 1 and max_peers")
	}
	if cfg.MaxPlayers > protocol.MaxRosterPlayers {
		return fmt.Errorf("max_players must not exceed %d", protocol.MaxRosterPlayers)
	}
	if cfg.JoinTimeoutMS < 1 || cfg.JoinTimeoutMS > int64(math.MaxInt64/time.Millisecond) {
		return errors.New("join_timeout_ms must be positive and fit in time.Duration")
	}
	if minimum := protocol.MinimumRoomFrameBytes(cfg.MaxPlayers); cfg.MaxFrameBytes < minimum {
		return fmt.Errorf("max_frame_bytes must be at least %d for this room", minimum)
	}
	return nil
}
