// 进程内消息总线与邮件/礼物公共逻辑

package messaging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const (
	giftTTLSeconds     = 30 * 24 * 3600
	emptyBuffSign      = "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945"
	messageQueueLength = 1000
)

// messageBus is process-local chat delivery. It intentionally does not persist:
// the legacy websocket queue had the same lifetime, and the HTTP poll drains it.
type messageBus struct {
	mu     sync.Mutex
	queues map[string][]Object
}

// 创建消息总线
func newMessageBus() *messageBus {
	return &messageBus{queues: map[string][]Object{}}
}

// 推送消息
func (b *messageBus) push(userID string, message Object) {
	b.mu.Lock()
	defer b.mu.Unlock()
	queue := append(b.queues[userID], message)
	if len(queue) > messageQueueLength {
		queue = queue[len(queue)-messageQueueLength:]
	}
	b.queues[userID] = queue
}

// 取出并清空消息
func (b *messageBus) drain(userID string) []Object {
	b.mu.Lock()
	defer b.mu.Unlock()
	queue := b.queues[userID]
	delete(b.queues, userID)
	return queue
}

// 判断消息队列是否为空
func (b *messageBus) empty(userID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.queues[userID]) == 0
}

// 广播聊天消息
func (a *Service) broadcastChat(entry Object) error {
	ids, err := a.Store.UserIDs(context.Background())
	if err != nil {
		return err
	}
	for _, id := range ids {
		a.bus.push(id, entry)
	}
	return nil
}

// 推送聊天消息
func (a *Service) pushChat(userID string, entry Object) {
	a.bus.push(userID, entry)
}

var giftEnum = []string{"", "gift_rainbow", "gift", "gift_wax", "gift_ap", "social_feed_reward", "gift_heart_wax", "gift_hearts", "gift_scandle"}

type giftSpec struct {
	payType, recvType string
	pay, recv         int64
}

var giftSpecs = map[string]giftSpec{
	"gift_heart_wax": {"", "", 0, 5},
	"gift_wax":       {"", "", 0, 5},
	"gift_hearts":    {"candles", "heart", 3, 1},
	"gift":           {"candles", "heart", 3, 1},
	"gift_scandle":   {"", "season_candle", 0, 1},
	"gift_ap":        {"", "", 0, 0},
	"gift_rainbow":   {"", "", 0, 0},
}

// 规范化礼物类型
func normalizeGiftType(v any) string {
	switch x := v.(type) {
	case bool:
		return ""
	case json.Number:
		n, err := x.Int64()
		if err != nil || n < 0 || int(n) >= len(giftEnum) {
			return ""
		}
		return giftEnum[n]
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return ""
		}
		if n, err := json.Number(s).Int64(); err == nil {
			if n >= 0 && int(n) < len(giftEnum) {
				return giftEnum[n]
			}
			return ""
		}
		if strings.HasPrefix(s, "-") {
			return ""
		}
		if _, ok := giftSpecs[s]; ok {
			return s
		}
		return ""
	default:
		return ""
	}
}

var giftIDState struct {
	sync.Mutex
	last int64
}

// 生成礼物 ID
func newGiftID() int64 {
	giftIDState.Lock()
	defer giftIDState.Unlock()
	value := time.Now().UnixMilli()*1000 + giftIDState.last%1000
	if value <= giftIDState.last {
		value = giftIDState.last + 1
	}
	giftIDState.last = value
	return value
}

// 转换礼物 ID
func giftIDInt(g repository.FreeGift) int64 {
	if n, err := json.Number(g.GiftID).Int64(); err == nil {
		return n
	}
	return g.ID
}

// 构造已发送礼物响应
func sentGiftEntry(g repository.FreeGift) Object {
	spec := giftSpecs[g.GiftType]
	return Object{"currency_count": spec.pay, "currency_type": spec.payType, "gift_id": giftIDInt(g), "msg_id": giftIDInt(g), "recvd": g.Received, "to_id": g.ReceiverID, "type": g.GiftType}
}

// 构造已接收礼物响应
func recvdGiftEntry(g repository.FreeGift) Object {
	spec := giftSpecs[g.GiftType]
	return Object{"from_id": g.SenderID, "gift_id": giftIDInt(g), "msg_id": giftIDInt(g), "recvd": g.Received, "to_currency_count": spec.recv, "to_currency_type": spec.recvType, "type": g.GiftType}
}

// 构造礼物消息响应
func recvdMessageEntry(g repository.FreeGift) Object {
	spec := giftSpecs[g.GiftType]
	return Object{"currency_count": spec.recv, "currency_type": spec.recvType, "from_id": g.SenderID, "gift_id": giftIDInt(g), "msg_id": giftIDInt(g), "raw_message": nil, "type": g.GiftType}
}

type mailMessage struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	ExpiresDays int    `json:"expires_days"`
	MinVersion  any    `json:"min_version"`
	MaxVersion  any    `json:"max_version"`
	Attachments []any  `json:"attachments"`
}

type mailboxConfig struct {
	FromID string        `json:"from_id"`
	Mails  []mailMessage `json:"mails"`
}

// 读取邮箱配置
func (a *Service) mailbox() mailboxConfig {
	fallback := mailboxConfig{FromID: "44e9f7e2-c9fb-4328-87d4-3544221856ce", Mails: []mailMessage{}}
	if a.Cfg.GameConfigDir == "" {
		return fallback
	}
	raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "mailbox.json"))
	if err != nil {
		return fallback
	}
	var cfg mailboxConfig
	if json.Unmarshal(raw, &cfg) != nil {
		return fallback
	}
	if cfg.FromID == "" {
		cfg.FromID = fallback.FromID
	}
	if cfg.Mails == nil {
		cfg.Mails = []mailMessage{}
	}
	return cfg
}

// 解析客户端版本
func versionBound(v any) ([3]int, bool) {
	var text string
	switch x := v.(type) {
	case string:
		text = x
	case json.Number:
		text = x.String()
	case float64:
		text = strconv.Itoa(int(x))
	default:
		return [3]int{}, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return [3]int{}, false
	}
	parts := strings.Split(text, ".")
	out := [3]int{}
	for i := 0; i < 3 && i < len(parts); i++ {
		var n int
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				return [3]int{}, false
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = n
	}
	return out, true
}

// 判断版本不低于
func versionAtLeast(v [3]int, min [3]int) bool {
	for i := range v {
		if v[i] > min[i] {
			return true
		}
		if v[i] < min[i] {
			return false
		}
	}
	return true
}

// 判断版本不高于
func versionAtMost(v [3]int, max [3]int) bool {
	for i := range v {
		if v[i] < max[i] {
			return true
		}
		if v[i] > max[i] {
			return false
		}
	}
	return true
}

// 判断邮件是否可见
func mailVisible(mail mailMessage, version [3]int, known bool) bool {
	min, hasMin := versionBound(mail.MinVersion)
	max, hasMax := versionBound(mail.MaxVersion)
	if !known {
		return !hasMin && !hasMax
	}
	if hasMin && !versionAtLeast(version, min) {
		return false
	}
	if hasMax && !versionAtMost(version, max) {
		return false
	}
	return true
}

// 构造邮件响应
func mailPayload(mail mailMessage, fromID, userID string, stamp int64, detail bool) Object {
	days := mail.ExpiresDays
	if days == 0 {
		days = 30
	}
	headers := Object{"id": mail.ID, "type": mail.Type, "from": fromID, "to": userID, "sent_at": stamp, "received_at": stamp, "expires_at": stamp + int64(days)*86400}
	if len(mail.Attachments) > 0 {
		headers["has_attachments"] = true
	}
	subject := mail.Subject
	properties := Object{"HasCurrencies": false, "HasLinks": false, "HasUnlocks": false}
	for _, raw := range mail.Attachments {
		attachment, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		switch text(attachment, "type") {
		case "currency":
			properties["HasCurrencies"] = true
		case "unlock":
			properties["HasUnlocks"] = true
		}
	}
	payload := Object{"headers": headers, "subject": subject, "properties": properties}
	if detail {
		payload["body"] = mail.Body
		if len(mail.Attachments) > 0 {
			payload["attachments"] = mail.Attachments
		}
	}
	return payload
}

// 排序去重
func sortedUnique(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := []int64{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// 把整数列表转 JSON
func intsJSON(ids []int64) string {
	encoded, _ := json.Marshal(ids)
	return string(encoded)
}
