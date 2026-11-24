// 太平洋日期跨夏令时测试

package service

import (
	"testing"
	"time"
)

func TestTask4PacificDayAcrossDaylightSaving(t *testing.T) {
	for _, tc := range []struct {
		utc  time.Time
		want string
	}{
		{time.Date(2026, 3, 8, 7, 59, 0, 0, time.UTC), "2026-03-07"},
		{time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC), "2026-03-08"},
		{time.Date(2026, 3, 9, 6, 59, 0, 0, time.UTC), "2026-03-08"},
		{time.Date(2026, 3, 9, 7, 0, 0, 0, time.UTC), "2026-03-09"},
		{time.Date(2026, 11, 1, 6, 59, 0, 0, time.UTC), "2026-10-31"},
		{time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC), "2026-11-01"},
		{time.Date(2026, 11, 2, 7, 59, 0, 0, time.UTC), "2026-11-01"},
		{time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC), "2026-11-02"},
	} {
		if day := pacificDay(tc.utc.Unix()); day != tc.want {
			t.Fatalf("%s: day=%s want=%s", tc.utc, day, tc.want)
		}
	}
}
