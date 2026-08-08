//go:build windows

package tray

import (
	"encoding/binary"
	"testing"
)

func TestIconICOIncludesMultipleSizes(t *testing.T) {
	icon := IconICO()
	wantSizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	if len(icon) < 6+16*len(wantSizes) {
		t.Fatalf("icon too small: %d", len(icon))
	}
	if got := binary.LittleEndian.Uint16(icon[0:2]); got != 0 {
		t.Fatalf("reserved = %d", got)
	}
	if got := binary.LittleEndian.Uint16(icon[2:4]); got != 1 {
		t.Fatalf("type = %d", got)
	}
	if got := int(binary.LittleEndian.Uint16(icon[4:6])); got != len(wantSizes) {
		t.Fatalf("count = %d", got)
	}
	for i, want := range wantSizes {
		base := 6 + i*16
		gotWidth := int(icon[base])
		gotHeight := int(icon[base+1])
		if gotWidth == 0 {
			gotWidth = 256
		}
		if gotHeight == 0 {
			gotHeight = 256
		}
		if gotWidth != want {
			got := gotWidth
			t.Fatalf("entry %d width = %d, want %d", i, got, want)
		}
		if gotHeight != want {
			got := gotHeight
			t.Fatalf("entry %d height = %d, want %d", i, got, want)
		}
		if bitCount := binary.LittleEndian.Uint16(icon[base+6 : base+8]); bitCount != 32 {
			t.Fatalf("entry %d bit count = %d", i, bitCount)
		}
	}
}

func TestFormatStatusLine(t *testing.T) {
	tests := []struct {
		name string
		in   Status
		want string
	}{
		{name: "healthy", in: Status{Healthy: true}, want: "运行中: localhost"},
		{name: "cooling keys", in: Status{Healthy: true, FailedKeys: 2}, want: "运行中: localhost (2 个 key 冷却)"},
		{name: "setup required", in: Status{Healthy: true, Readiness: "unconfigured"}, want: "待配置: localhost"},
		{name: "unavailable", in: Status{Healthy: true, Readiness: "unavailable"}, want: "不可用: localhost"},
		{name: "degraded", in: Status{Healthy: true, Readiness: "degraded", FailedKeys: 2}, want: "部分可用: localhost (2 个 key 冷却)"},
		{name: "unhealthy", in: Status{Healthy: false}, want: "状态异常: localhost"},
		{name: "error", in: Status{Healthy: true, Error: "db failed"}, want: "状态异常: 无法读取"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStatusLine(tt.in); got != tt.want {
				t.Fatalf("formatStatusLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
