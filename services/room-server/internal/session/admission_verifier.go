package session

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

var ErrAdmissionRejected = errors.New("API rejected the room admission")

type VerifiedAdmission struct {
	AccountID string
	ExpiresAt time.Time
}

type AdmissionVerifier struct {
	endpoint string
	bearer   string
	client   *http.Client
}

func NewAdmissionVerifier(baseURL, bearer string) (*AdmissionVerifier, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil ||
		(endpoint.Path != "" && endpoint.Path != "/") ||
		endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("room API URL must be an HTTP origin")
	}
	ip := net.ParseIP(endpoint.Hostname())
	if endpoint.Scheme != "https" && (endpoint.Scheme != "http" || ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("room API requires HTTPS or an HTTP loopback IP")
	}
	if len(bearer) < 32 {
		return nil, errors.New("room admission service credential must contain at least 32 bytes")
	}
	endpoint.Path = "/internal/v1/room/admissions/verify"
	return &AdmissionVerifier{
		endpoint: endpoint.String(),
		bearer:   bearer,
		client: &http.Client{
			Timeout: 2 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (v *AdmissionVerifier) Verify(ctx context.Context, roomURI string, proof protocol.AdmissionProof) (VerifiedAdmission, error) {
	body, err := json.Marshal(map[string]string{
		"account_id": proof.AccountID,
		"room_uri":   roomURI,
		"signature":  hex.EncodeToString(proof.Signature[:]),
	})
	if err != nil {
		return VerifiedAdmission{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, bytes.NewReader(body))
	if err != nil {
		return VerifiedAdmission{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.bearer)

	resp, err := v.client.Do(req)
	if err != nil {
		return VerifiedAdmission{}, fmt.Errorf("call room admission API: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return VerifiedAdmission{}, ErrAdmissionRejected
	default:
		return VerifiedAdmission{}, fmt.Errorf("room admission API returned status %d", resp.StatusCode)
	}

	var result struct {
		AccountID string `json:"account_id"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return VerifiedAdmission{}, errors.New("invalid room admission API response")
	}
	expires := time.Unix(result.ExpiresAt, 0)
	if result.AccountID == "" || result.AccountID != proof.AccountID || !time.Now().Before(expires) {
		return VerifiedAdmission{}, errors.New("room admission API returned mismatched or expired identity")
	}
	return VerifiedAdmission{AccountID: result.AccountID, ExpiresAt: expires}, nil
}
