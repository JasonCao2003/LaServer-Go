//go:build windows && amd64

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/app"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/room"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("room-server", flag.ContinueOnError)
	cfgPath := flags.String("config", "config/realtime.json", "room server JSON configuration")
	dllPath := flags.String("enet-bridge", "", "native SkyENet bridge DLL")
	apiURL := flags.String("admission-api", "", "API server origin for admission verification")
	advertise := flags.String("announce", "", "public IPv4 UDP address supplied by API discovery")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *dllPath == "" {
		return errors.New("provide -enet-bridge and omit positional arguments")
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	if cfg.MaxPlayers > 255 {
		return errors.New("this room profile supports at most 255 players")
	}
	bind, err := netip.ParseAddrPort(cfg.UDPListen)
	if err != nil || !bind.Addr().Is4() {
		return errors.New("this SkyENet build requires an IPv4 UDP listen address")
	}
	announced, err := netip.ParseAddrPort(*advertise)
	if err != nil || !announced.Addr().Is4() || announced.Addr().IsUnspecified() || announced.Port() == 0 {
		return errors.New("provide a valid IPv4:port address with -announce")
	}
	dll, err := filepath.Abs(*dllPath)
	if err != nil {
		return err
	}
	verifier, err := session.NewAdmissionVerifier(*apiURL, os.Getenv("SKYLA_ROOM_ADMISSION_BEARER"))
	if err != nil {
		return err
	}
	return app.Run(ctx, cfg, func(cfg config.Config) (app.Transport, error) {
		host, err := transport.OpenNative(dll, bind, cfg.MaxPeers, cfg.MaxFrameBytes)
		if err != nil {
			return nil, err
		}
		log.Printf("listening on UDP %s, announcing %s", bind, announced)
		return room.NewServer(host, cfg, announced.String(), verifier), nil
	})
}
