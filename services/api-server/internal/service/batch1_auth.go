package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) ResourceAuthToken(_ Request, _ Object) (Object, error) {
	if a.cfg.ResourceAuthToken == "" {
		return nil, fail(503, "Resource authentication unavailable")
	}
	return Object{"host": "sky.resources.cdn.thatgamecompany.com", "prefix": "", "status": "OK", "token": a.cfg.ResourceAuthToken}, nil
}

func randomHex(bytesCount int) (string, error) {
	b := make([]byte, bytesCount)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func signedJWT(secret string, claims Object) (string, error) {
	header := Object{"alg": "HS256", "typ": "JWT"}
	h, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	c, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encode := base64.RawURLEncoding.EncodeToString
	message := encode(h) + "." + encode(c)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return message + "." + encode(mac.Sum(nil)), nil
}

func (a *Service) StarwatchAuth(r Request, m Object) (Object, error) {
	session := text(m, "sessionToken")
	var userID string
	err := a.store.QueryRowContext(r.Context, repository.UserIDBySession, session).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) || session == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "not_found"}}
	}
	if err != nil {
		return nil, err
	}
	if a.cfg.StarwatchJWTSecret == "" || a.cfg.StarwatchBrokerURL == "" {
		return nil, fail(503, "Starwatch authentication unavailable")
	}
	device := text(m, "deviceId")
	if strings.ContainsAny(device, "+#/\x00\r\n") || strings.ContainsAny(userID, "+#/\x00\r\n") {
		return nil, fail(400, "Invalid device or user identifier")
	}
	randomID, err := randomHex(11)
	if err != nil {
		return nil, err
	}
	clientID := "sky-beta-game-" + randomID
	username := "user::" + randomID
	base := "sky/beta/device/" + device + "/user/" + userID
	live := "sky/live/device/" + device + "/user/" + userID
	nowTime := time.Now()
	claims := Object{"client_id": clientID, "username": username, "acl": Object{"pub": []string{base + "/event/+", live + "/remote/prompt"}, "sub": []string{live + "/remote/submit"}}, "iat": nowTime.Unix(), "iss": "sw-auth", "aud": "sw-broker", "exp": nowTime.Add(24 * time.Hour).Unix(), "jit": randomID}
	jwt, err := signedJWT(a.cfg.StarwatchJWTSecret, claims)
	if err != nil {
		return nil, err
	}
	sessionID, err := uuid()
	if err != nil {
		return nil, err
	}
	millis := nowTime.UnixMilli()
	return Object{"mqtt": Object{"clientId": clientID, "username": username, "password": jwt, "brokerUrl": a.cfg.StarwatchBrokerURL}, "settings": Object{"enableEventPublishing": true, "blockedEvents": []string{"account_resource_waiting", "candle_accepted_cost", "checkpoint", "dayhm_manta_count_03", "debug_trace", "find_prev_game_request", "match_joined", "net_game_connect_attempt", "net_game_move_timeout", "sync_user_data", "*_loaded", "loaded_*", "player_nearby", "player_nearby_left"}}, "acls": Object{"publish": []string{base + "/event/+", base + "/remote/prompt", base + "/challenge/response"}, "subscribe": []string{base + "/remote/submit", base + "/challenge/submit"}}, "ownerId": "sky::beta::game::" + randomID, "isJwt": true, "expiresSeconds": 86400, "starwatchSessionId": sessionID, "serverTime": Object{"requestStartMillis": millis, "requestEndMillis": millis + 15}}, nil
}
