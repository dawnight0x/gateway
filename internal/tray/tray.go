//go:build windows

package tray

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/getlantern/systray"

	"local-ai-gateway/internal/desktop"
)

type Options struct {
	AdminURL        string
	AdminOpenURL    string
	DataDir         string
	LogPath         string
	Status          func() Status
	OpenAIConfig    string
	AnthropicConfig string
	GeminiConfig    string
	Restart         func() error
	Shutdown        func()
}

type Status struct {
	Healthy       bool
	ActiveKeys    int
	FailedKeys    int
	TodayRequests int
	TodayTokens   int
	Error         string
}

func Supported() bool {
	return true
}

func Run(opts Options) {
	systray.Run(func() {
		slog.Info("system tray initialized")
		systray.SetIcon(IconICO())
		systray.SetTitle("AI Gateway")
		systray.SetTooltip("AI Gateway 正在启动")

		status := systray.AddMenuItem("正在读取状态...", "本地网关状态")
		status.Disable()
		keysStatus := systray.AddMenuItem("Keys: -", "可用 key / 冷却 key")
		keysStatus.Disable()
		usageStatus := systray.AddMenuItem("Today: -", "今日请求和 token")
		usageStatus.Disable()
		systray.AddSeparator()
		openAdmin := systray.AddMenuItem("打开管理后台", "打开本地管理网页")
		copyAdmin := systray.AddMenuItem("复制管理地址", "复制管理后台地址")
		copyOpenAI := systray.AddMenuItem("复制 OpenAI 配置", "复制 OpenAI-compatible 环境变量")
		copyAnthropic := systray.AddMenuItem("复制 Anthropic 配置", "复制 Anthropic-compatible 环境变量")
		copyGemini := systray.AddMenuItem("复制 Gemini 配置", "复制 Gemini-compatible 环境变量")
		systray.AddSeparator()
		openDataDir := systray.AddMenuItem("打开数据目录", "打开数据库、日志和锁文件目录")
		openLog := systray.AddMenuItem("打开日志文件", "打开本地运行日志")
		systray.AddSeparator()
		autostart := systray.AddMenuItemCheckbox("开机自动启动", "登录 Windows 后自动启动本地网关", false)
		restart := systray.AddMenuItem("重启网关", "重启本地网关进程")
		exit := systray.AddMenuItem("退出网关", "关闭网关进程并移除托盘图标")

		refreshAutostartMenu(autostart)
		refreshTrayStatus(opts, status, keysStatus, usageStatus)
		statusTicker := time.NewTicker(15 * time.Second)

		go func() {
			defer statusTicker.Stop()
			for {
				select {
				case <-statusTicker.C:
					refreshTrayStatus(opts, status, keysStatus, usageStatus)
				case <-openAdmin.ClickedCh:
					target := opts.AdminOpenURL
					if target == "" {
						target = opts.AdminURL
					}
					if err := desktop.OpenBrowser(target); err != nil {
						slog.Warn("open browser failed", "error", err)
					}
				case <-copyAdmin.ClickedCh:
					if err := desktop.CopyText(opts.AdminURL); err != nil {
						slog.Warn("copy admin url failed", "error", err)
					}
				case <-copyOpenAI.ClickedCh:
					if err := desktop.CopyText(opts.OpenAIConfig); err != nil {
						slog.Warn("copy openai config failed", "error", err)
					}
				case <-copyAnthropic.ClickedCh:
					if err := desktop.CopyText(opts.AnthropicConfig); err != nil {
						slog.Warn("copy anthropic config failed", "error", err)
					}
				case <-copyGemini.ClickedCh:
					if err := desktop.CopyText(opts.GeminiConfig); err != nil {
						slog.Warn("copy gemini config failed", "error", err)
					}
				case <-openDataDir.ClickedCh:
					if err := desktop.OpenPath(opts.DataDir); err != nil {
						slog.Warn("open data dir failed", "error", err)
					}
				case <-openLog.ClickedCh:
					if err := desktop.OpenPath(opts.LogPath); err != nil {
						slog.Warn("open log file failed", "error", err)
					}
				case <-autostart.ClickedCh:
					next := !autostart.Checked()
					if err := desktop.SetAutostartEnabled(next); err != nil {
						slog.Warn("toggle autostart failed", "enabled", next, "error", err)
					}
					refreshAutostartMenu(autostart)
				case <-restart.ClickedCh:
					if err := restartAndQuit(opts.Restart, systray.Quit); err != nil {
						slog.Error("restart failed", "error", err)
						continue
					}
					return
				case <-exit.ClickedCh:
					opts.Shutdown()
					systray.Quit()
					return
				}
			}
		}()
	}, func() {
		slog.Info("system tray stopped")
		opts.Shutdown()
	})
}

func refreshTrayStatus(opts Options, status, keysStatus, usageStatus *systray.MenuItem) {
	st := Status{Healthy: true}
	if opts.Status != nil {
		st = opts.Status()
	}
	text := formatStatusLine(st)
	status.SetTitle(text)
	if st.Error != "" {
		systray.SetTooltip("AI Gateway 状态读取失败: " + st.Error)
		keysStatus.SetTitle("Keys: 状态未知")
		usageStatus.SetTitle("Today: 状态未知")
		return
	}
	keysStatus.SetTitle(fmt.Sprintf("Keys: %d 可用 / %d 冷却", st.ActiveKeys, st.FailedKeys))
	usageStatus.SetTitle(fmt.Sprintf("Today: %d 请求 / %d tokens", st.TodayRequests, st.TodayTokens))
	systray.SetTooltip(fmt.Sprintf("AI Gateway | %d 可用 key | %d 冷却 | 今日 %d 请求", st.ActiveKeys, st.FailedKeys, st.TodayRequests))
}

func refreshAutostartMenu(item *systray.MenuItem) {
	enabled, err := desktop.IsAutostartEnabled()
	if err != nil {
		slog.Warn("read autostart failed", "error", err)
		item.SetTitle("开机自动启动: 状态未知")
		item.Uncheck()
		return
	}
	item.SetTitle("开机自动启动")
	if enabled {
		item.Check()
		return
	}
	item.Uncheck()
}

func formatStatusLine(st Status) string {
	if st.Error != "" {
		return "状态异常: 无法读取"
	}
	if !st.Healthy {
		return "状态异常: localhost"
	}
	if st.FailedKeys > 0 {
		return fmt.Sprintf("运行中: localhost (%d 个 key 冷却)", st.FailedKeys)
	}
	return "运行中: localhost"
}

func Quit() {
	systray.Quit()
}

func IconICO() []byte {
	sizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	images := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		images = append(images, iconDIB(size))
	}

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))

	offset := 6 + 16*len(sizes)
	for i, size := range sizes {
		buf.WriteByte(byte(size))
		buf.WriteByte(byte(size))
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(images[i])))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(images[i])
	}
	for _, img := range images {
		buf.Write(img)
	}
	return buf.Bytes()
}

func iconDIB(size int) []byte {
	maskStride := ((size + 31) / 32) * 4
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.LittleEndian, uint32(40))
	_ = binary.Write(&buf, binary.LittleEndian, int32(size))
	_ = binary.Write(&buf, binary.LittleEndian, int32(size*2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(size*size*4))
	_ = binary.Write(&buf, binary.LittleEndian, int32(0))
	_ = binary.Write(&buf, binary.LittleEndian, int32(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))

	pixels := make([]rgba, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			pixels[y*size+x] = iconPixel(size, x, y)
		}
	}

	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			p := pixels[y*size+x]
			buf.Write([]byte{p.b, p.g, p.r, p.a})
		}
	}
	buf.Write(make([]byte, maskStride*size))
	return buf.Bytes()
}

type rgba struct {
	r byte
	g byte
	b byte
	a byte
}

func iconPixel(size, x, y int) rgba {
	px := float64(x) + 0.5
	py := float64(y) + 0.5
	s := float64(size)
	inset := math.Max(1, s*0.055)
	radius := s * 0.22
	dist := roundedRectDistance(px, py, s/2, s/2, s/2-inset, s/2-inset, radius)
	alpha := smoothAlpha(dist)
	if alpha <= 0 {
		return rgba{}
	}

	fill := rgba{r: 49, g: 91, b: 214, a: alpha}
	if dist > -math.Max(1.15, s*0.075) {
		fill = mix(fill, rgba{r: 255, g: 255, b: 255, a: alpha}, 0.20)
	}
	fill.a = alpha

	symbol := gatewaySymbol(size, px, py)
	return over(symbol, fill)
}

func gatewaySymbol(size int, x, y float64) rgba {
	s := float64(size)
	center := point{s * 0.50, s * 0.51}
	outer := math.Min(s*0.31, s/2-2)
	stroke := math.Max(2.1, s*0.115)
	if size <= 20 {
		stroke = math.Max(2.25, s*0.135)
	}

	d := math.Hypot(x-center.x, y-center.y)
	ring := clamp01((stroke/2 + 0.65 - math.Abs(d-outer)) / 1.25)

	// Open the right side to read as a clean gateway "G" at tray size.
	angle := math.Atan2(y-center.y, x-center.x)
	gap := math.Abs(angle) < 0.48 && x > center.x+s*0.13
	if gap {
		ring = 0
	}

	bar := roundedRectCoverage(
		x, y,
		center.x+s*0.04,
		center.y+s*0.01,
		s*0.23,
		stroke*0.52,
		stroke*0.27,
	)
	notch := roundedRectCoverage(
		x, y,
		center.x+s*0.24,
		center.y-stroke*0.34,
		s*0.13,
		stroke*0.50,
		stroke*0.22,
	)
	cut := roundedRectCoverage(
		x, y,
		center.x+s*0.21,
		center.y+s*0.145,
		s*0.15,
		stroke*0.38,
		stroke*0.18,
	)

	a := math.Max(ring, math.Min(1, bar+notch))
	a = math.Max(0, a-cut*0.80)
	if a <= 0 {
		return rgba{}
	}
	return rgba{r: 255, g: 255, b: 255, a: byte(math.Round(255 * clamp01(a)))}
}

type point struct {
	x float64
	y float64
}

func roundedRectDistance(x, y, cx, cy, hx, hy, r float64) float64 {
	qx := math.Abs(x-cx) - hx + r
	qy := math.Abs(y-cy) - hy + r
	outside := math.Hypot(math.Max(qx, 0), math.Max(qy, 0))
	inside := math.Min(math.Max(qx, qy), 0)
	return outside + inside - r
}

func smoothAlpha(distance float64) byte {
	if distance <= -0.65 {
		return 255
	}
	if distance >= 0.65 {
		return 0
	}
	return byte(math.Round((0.65 - distance) / 1.3 * 255))
}

func lineCoverage(x, y float64, a, b point, width float64) float64 {
	vx := b.x - a.x
	vy := b.y - a.y
	wx := x - a.x
	wy := y - a.y
	length2 := vx*vx + vy*vy
	if length2 == 0 {
		return 0
	}
	t := math.Max(0, math.Min(1, (wx*vx+wy*vy)/length2))
	projX := a.x + t*vx
	projY := a.y + t*vy
	d := math.Hypot(x-projX, y-projY)
	return clamp01((width + 0.55 - d) / 1.1)
}

func circleCoverage(x, y float64, center point, radius float64) float64 {
	d := math.Hypot(x-center.x, y-center.y)
	return clamp01((radius + 0.55 - d) / 1.1)
}

func roundedRectCoverage(x, y, cx, cy, hx, hy, r float64) float64 {
	return clamp01((0.65 - roundedRectDistance(x, y, cx, cy, hx, hy, r)) / 1.3)
}

func mix(a, b rgba, t float64) rgba {
	t = clamp01(t)
	return rgba{
		r: byte(math.Round(float64(a.r)*(1-t) + float64(b.r)*t)),
		g: byte(math.Round(float64(a.g)*(1-t) + float64(b.g)*t)),
		b: byte(math.Round(float64(a.b)*(1-t) + float64(b.b)*t)),
		a: byte(math.Round(float64(a.a)*(1-t) + float64(b.a)*t)),
	}
}

func over(top, bottom rgba) rgba {
	if top.a == 0 {
		return bottom
	}
	if bottom.a == 0 || top.a == 255 {
		return top
	}
	ta := float64(top.a) / 255
	ba := float64(bottom.a) / 255
	outA := ta + ba*(1-ta)
	if outA <= 0 {
		return rgba{}
	}
	return rgba{
		r: byte(math.Round((float64(top.r)*ta + float64(bottom.r)*ba*(1-ta)) / outA)),
		g: byte(math.Round((float64(top.g)*ta + float64(bottom.g)*ba*(1-ta)) / outA)),
		b: byte(math.Round((float64(top.b)*ta + float64(bottom.b)*ba*(1-ta)) / outA)),
		a: byte(math.Round(outA * 255)),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
