// 实例级 IP 封禁规则。
package ipban

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const maxRules = 1024

var (
	ErrInvalid = errors.New("invalid IP address or network")
	ErrFull    = errors.New("IP ban list is full")
)

type Rules struct {
	directory string
	mu        sync.RWMutex
	entries   []netip.Prefix
	loadErr   error
}

func Open(directory string) *Rules {
	r := &Rules{directory: directory}
	if directory == "" {
		return r
	}
	entries, err := os.ReadFile(filepath.Join(directory, "banned_ips.json"))
	if errors.Is(err, os.ErrNotExist) {
		return r
	}
	if err == nil {
		r.entries, err = decode(entries)
	}
	r.loadErr = err
	return r
}

func decode(raw []byte) ([]netip.Prefix, error) {
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if len(values) > maxRules {
		return nil, ErrFull
	}
	entries := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := parse(value)
		if err != nil {
			return nil, fmt.Errorf("load IP bans: %w", err)
		}
		entries = append(entries, prefix)
	}
	return entries, nil
}

func parse(value string) (netip.Prefix, error) {
	value = strings.TrimSpace(value)
	if address, err := netip.ParseAddr(value); err == nil && address.Zone() == "" {
		address = address.Unmap()
		return netip.PrefixFrom(address, address.BitLen()), nil
	}
	if prefix, err := netip.ParsePrefix(value); err == nil && !prefix.Addr().Is4In6() {
		return prefix.Masked(), nil
	}
	return netip.Prefix{}, ErrInvalid
}

func (r *Rules) Match(ip netip.Addr) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.loadErr != nil {
		return false, r.loadErr
	}
	ip = ip.WithZone("").Unmap()
	for _, prefix := range r.entries {
		if prefix.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

func (r *Rules) List() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return display(r.entries), r.loadErr
}

func (r *Rules) Change(value string, add bool) ([]string, error) {
	prefix, err := parse(value)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	index := slices.Index(r.entries, prefix)
	if (add && index >= 0) || (!add && index < 0) {
		return display(r.entries), nil
	}
	entries := slices.Clone(r.entries)
	if add {
		if len(entries) >= maxRules {
			return nil, ErrFull
		}
		entries = append(entries, prefix)
	} else {
		entries = slices.Delete(entries, index, index+1)
	}
	values := display(entries)
	if err := r.save(values); err != nil {
		return nil, err
	}
	r.entries = entries
	return values, nil
}

func display(entries []netip.Prefix) []string {
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Bits() == entry.Addr().BitLen() {
			values = append(values, entry.Addr().String())
		} else {
			values = append(values, entry.String())
		}
	}
	slices.Sort(values)
	return values
}

func (r *Rules) save(entries []string) error {
	if r.directory == "" {
		return errors.New("instance directory missing")
	}
	if err := os.MkdirAll(r.directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(r.directory, "banned-ips-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = json.NewEncoder(file).Encode(entries)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), filepath.Join(r.directory, "banned_ips.json"))
}
