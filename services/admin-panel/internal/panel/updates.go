package panel

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type updatePackage struct {
	Section, Revision string
	Value             json.RawMessage
}
type receiverState struct {
	sync.Mutex
	server        *http.Server
	address, last string
}

func (p *panelServer) receive(w http.ResponseWriter, r *http.Request) {
	key, err := base64.StdEncoding.DecodeString(os.Getenv("LASERVER_UPDATE_PUBLIC_KEY"))
	if err != nil || len(key) != ed25519.PublicKeySize {
		http.Error(w, "接收密钥未配置", 503)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256<<10))
	if err != nil {
		http.Error(w, "批次过大", 413)
		return
	}
	sig, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Update-Signature"))
	if err != nil || !ed25519.Verify(key, raw, sig) {
		http.Error(w, "签名无效", 403)
		return
	}
	var pack updatePackage
	valid := false
	if json.Unmarshal(raw, &pack) == nil && len(pack.Value) > 0 && len(pack.Revision) == 64 {
		for _, item := range operationSections {
			if item.Key == pack.Section {
				valid = true
			}
		}
	}
	if !valid {
		http.Error(w, "无效的配置批次", 400)
		return
	}
	digest := sha256.Sum256(raw)
	id := hex.EncodeToString(digest[:])
	state := &p.receiver
	state.Lock()
	defer state.Unlock()
	for _, suffix := range []string{"pending", "applied", "rejected"} {
		if _, err := os.Stat(filepath.Join(p.cfg.updateDir, id+"."+suffix)); err == nil {
			io.WriteString(w, id+" "+suffix)
			return
		}
	}
	f, err := os.CreateTemp(p.cfg.updateDir, ".receive-*")
	if err == nil {
		defer os.Remove(f.Name())
		_, err = f.Write(raw)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(f.Name(), filepath.Join(p.cfg.updateDir, id+".pending"))
		}
	}
	if err != nil {
		http.Error(w, "暂存失败", 500)
		return
	}
	state.last = "收到批次 " + id
	io.WriteString(w, id+" pending")
}
func (p *panelServer) updates(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", 303)
		return
	}
	state := &p.receiver
	state.Lock()
	defer state.Unlock()
	if r.Method == "POST" {
		if !p.parseLimitedPost(w, r, 8192) {
			return
		}
		if subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, "Forbidden", 403)
			return
		}
		var err error
		switch r.PostForm.Get("action") {
		case "start":
			if state.server == nil {
				addr := os.Getenv("LASERVER_UPDATE_LISTEN")
				if addr == "" {
					addr = "127.0.0.1:9527"
				}
				key, e := base64.StdEncoding.DecodeString(os.Getenv("LASERVER_UPDATE_PUBLIC_KEY"))
				if e != nil || len(key) != ed25519.PublicKeySize {
					err = errors.New("请配置 Ed25519 发布公钥")
					break
				}
				if err = os.MkdirAll(p.cfg.updateDir, 0700); err != nil {
					break
				}
				var l net.Listener
				l, err = net.Listen("tcp", addr)
				if err == nil {
					mux := http.NewServeMux()
					mux.HandleFunc("POST /updates", p.receive)
					srv := &http.Server{Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 15 * time.Second}
					state.server = srv
					state.address = l.Addr().String()
					go func() {
						e := srv.Serve(l)
						state.Lock()
						defer state.Unlock()
						if state.server == srv {
							state.server = nil
							state.address = ""
							state.last = e.Error()
						}
					}()
				}
			}
		case "stop":
			if state.server != nil {
				err = state.server.Close()
				state.server = nil
				state.address = ""
			}
		case "apply", "reject":
			id := r.PostForm.Get("id")
			digest, e := hex.DecodeString(id)
			if e != nil || len(digest) != 32 {
				http.Error(w, "批次 ID 无效", 400)
				return
			}
			base := filepath.Join(p.cfg.updateDir, id)
			var raw []byte
			raw, err = os.ReadFile(base + ".pending")
			if err != nil {
				break
			}
			suffix := ".rejected"
			if r.PostForm.Get("action") == "apply" {
				var pack updatePackage
				sum := sha256.Sum256(raw)
				if hex.EncodeToString(sum[:]) != id || json.Unmarshal(raw, &pack) != nil {
					err = errors.New("批次内容已变化")
					break
				}
				var result []byte
				result, err = p.callAdmin(r.Context(), "/admin/api/operations", map[string]string{"action": "save", "section": pack.Section, "revision": pack.Revision, "value": string(pack.Value)})
				var ack struct{ OK bool }
				if err == nil && (json.Unmarshal(result, &ack) != nil || !ack.OK) {
					err = errors.New("API 未确认应用")
				}
				if err != nil {
					break
				}
				suffix = ".applied"
			}
			err = os.Rename(base+".pending", base+suffix)
			if err == nil {
				state.last = id + suffix
			}
		default:
			http.Error(w, "操作无效", 400)
			return
		}
		if err != nil {
			state.last = err.Error()
			http.Error(w, state.last, 409)
			return
		}
		http.Redirect(w, r, "/updates", 303)
		return
	}
	type batch struct{ ID, Status, Preview string }
	var batches []batch
	entries, err := os.ReadDir(p.cfg.updateDir)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "读取暂存区失败", 500)
		return
	}
	for _, entry := range entries {
		id, ext, _ := strings.Cut(entry.Name(), ".")
		if len(id) != 64 {
			continue
		}
		b := batch{ID: id, Status: ext}
		if ext == "pending" {
			raw, e := os.ReadFile(filepath.Join(p.cfg.updateDir, entry.Name()))
			if e == nil {
				b.Preview = string(raw)
			}
		}
		batches = append(batches, b)
	}
	p.render(w, "updates.html", 200, struct {
		CSRF, Address, Last string
		Batches             []batch
	}{session.csrf, state.address, state.last, batches})
}
