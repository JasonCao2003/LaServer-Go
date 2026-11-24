// 会话注册表测试

package session

import (
	"errors"
	"testing"
	"time"
)

func TestReconnectDoesNotAcceptStaleAdmission(t *testing.T) {
	r := NewRegistry(4, 2, time.Second)
	now := time.Unix(1, 0)
	first, err := r.Connect(1, now)
	if err != nil || r.StartJoin(first, now) != nil {
		t.Fatal("could not begin first admission")
	}
	r.Disconnect(first)
	second, err := r.Connect(1, now)
	if err != nil || r.StartJoin(second, now) != nil {
		t.Fatal("could not begin second admission")
	}
	if err := r.Admit(first, "account", now); !errors.Is(err, ErrUnknown) {
		t.Fatalf("stale admission error = %v", err)
	}
	if err := r.Admit(second, "account", now); err != nil {
		t.Fatal(err)
	}
	r.Disconnect(first)
	if !r.Online("account") {
		t.Fatal("stale disconnect removed current session")
	}
	r.Disconnect(second)
	if r.Online("account") {
		t.Fatal("disconnected account remains online")
	}
}

func TestRegistryAdmissionLimits(t *testing.T) {
	r := NewRegistry(2, 1, time.Second)
	now := time.Unix(1, 0)
	a, _ := r.Connect(1, now)
	b, _ := r.Connect(2, now)
	if _, err := r.Connect(3, now); !errors.Is(err, ErrFull) {
		t.Fatalf("peer capacity error = %v", err)
	}
	if err := r.StartJoin(a, now); err != nil {
		t.Fatal(err)
	}
	if err := r.StartJoin(a, now); !errors.Is(err, ErrState) {
		t.Fatalf("duplicate join error = %v", err)
	}
	if err := r.StartJoin(b, now); err != nil {
		t.Fatal(err)
	}
	if err := r.Admit(a, "", now); !errors.Is(err, ErrIdentity) {
		t.Fatalf("empty identity error = %v", err)
	}
	if err := r.Admit(a, "account", now); err != nil {
		t.Fatal(err)
	}
	if err := r.Admit(b, "account", now); !errors.Is(err, ErrDuplicateAccount) {
		t.Fatalf("duplicate account error = %v", err)
	}
	if err := r.Admit(b, "other", now); !errors.Is(err, ErrFull) {
		t.Fatalf("player capacity error = %v", err)
	}
}

func TestRegistryExpiresPendingAdmission(t *testing.T) {
	r := NewRegistry(2, 1, time.Second)
	now := time.Unix(1, 0)
	connected, _ := r.Connect(1, now)
	verifying, _ := r.Connect(2, now)
	if err := r.StartJoin(verifying, now); err != nil {
		t.Fatal(err)
	}
	deadline := now.Add(time.Second)
	if err := r.Admit(verifying, "account", deadline); !errors.Is(err, ErrExpired) {
		t.Fatalf("late admission error = %v", err)
	}
	expired := r.Expire(deadline)
	if len(expired) != 2 || !containsTicket(expired, connected) || !containsTicket(expired, verifying) {
		t.Fatalf("expired tickets = %+v", expired)
	}
	if r.Online("account") {
		t.Fatal("expired account became online")
	}
	if _, err := r.Connect(1, deadline); err != nil {
		t.Fatalf("expired peer was not released: %v", err)
	}
}

func containsTicket(tickets []Ticket, want Ticket) bool {
	for _, ticket := range tickets {
		if ticket == want {
			return true
		}
	}
	return false
}
