// JSON 输出兼容性测试

package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteKeepsLegacyUnlockNameAmpersandLiteral(t *testing.T) {
	w := httptest.NewRecorder()
	write(w, 200, object{"unlocks": []object{{"name": "reveal friend & family finish"}}})
	if strings.Contains(w.Body.String(), `\u0026`) {
		t.Fatalf("legacy client cannot decode escaped ampersand: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "reveal friend & family finish") {
		t.Fatalf("unlock name missing from response: %s", w.Body.String())
	}
}
