package config

import "testing"

func TestLegacyLoginModeRequiresExplicitPrivateListen(t *testing.T) {
	for _, listen := range []string{"", "0.0.0.0:8443", "[::]:8443", "8.8.8.8:8443", "example.com:8443"} {
		cfg := Config{LoginMode: "legacy_dev", Listen: listen, RoomHost: "127.0.0.1", RoomPort: 25565}
		if err := ValidateRuntime(cfg); err == nil {
			t.Errorf("legacy_dev accepted %q", listen)
		}
	}
	for _, listen := range []string{"127.0.0.1:8443", "192.168.1.10:8443"} {
		cfg := Config{LoginMode: "legacy_dev", Listen: listen, RoomHost: "127.0.0.1", RoomPort: 25565}
		if err := ValidateRuntime(cfg); err != nil {
			t.Errorf("legacy_dev rejected %q: %v", listen, err)
		}
	}
}

func TestLoginModeRejectsUnknownAndUnsafeAccountRecreation(t *testing.T) {
	for _, cfg := range []Config{
		{LoginMode: "unknown", RoomHost: "127.0.0.1", RoomPort: 25565},
		{RecreateMissingAccount: true, RoomHost: "127.0.0.1", RoomPort: 25565},
	} {
		if err := ValidateRuntime(cfg); err == nil {
			t.Errorf("accepted unsafe config: %+v", cfg)
		}
	}
}
