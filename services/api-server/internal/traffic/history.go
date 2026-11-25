package traffic

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Entry struct {
	Time              time.Time
	Request, Response string
}

var history struct {
	sync.Mutex
	rows        [128]Entry
	next, count int
}

func Add(request, response string) {
	history.Lock()
	defer history.Unlock()
	history.rows[history.next] = Entry{time.Now().UTC(), request, response}
	history.next = (history.next + 1) % len(history.rows)
	if history.count < len(history.rows) {
		history.count++
	}
}
func Handler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(r.Header.Get("X-Admin-Token"))) != 1 {
			http.Error(w, "Forbidden", 403)
			return
		}
		history.Lock()
		rows := make([]Entry, history.count)
		for i := range rows {
			rows[i] = history.rows[(history.next+len(history.rows)-1-i)%len(history.rows)]
		}
		history.Unlock()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	}
}
