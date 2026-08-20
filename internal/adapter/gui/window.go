package gui

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"

	"rlapi2mqtt/assets"
	"rlapi2mqtt/internal/service"
)

type Config struct {
	MQTTUrl      string
	MQTTUsername string
	MQTTPassword string
	RLAddress    string
	AutoConnect  bool
}

const (
	bannerHeight   = 40 // reserved vertical offset for controls below the banner
	bannerImageW   = 197
	bannerImageH   = 30
	bannerLeft     = 10
	bannerTop      = 5
	compactHeight  = 210 + bannerHeight
	expandedHeight = 430 + bannerHeight
)

type MainWindow struct {
	wnd            *ui.Main
	bannerDc       win.HDC // memory DC with the composited banner DIB selected (BitBlt source)
	bannerBmp      win.HBITMAP
	lblBroker      *ui.Static
	txtBroker      *ui.Edit
	lblUser        *ui.Static
	txtUser        *ui.Edit
	lblPass        *ui.Static
	txtPass        *ui.Edit
	lblGameWS      *ui.Static
	txtGameWS      *ui.Edit
	btnConnect     *ui.Button
	btnSaveConfig  *ui.Button
	lblMQTTStatus  *ui.Static
	lblRLStatus    *ui.Static
	chkAutoConnect *ui.CheckBox
	chkShowLogs    *ui.CheckBox
	chkLimitUpdate *ui.CheckBox
	lblLog         *ui.Static
	txtLog         *ui.Edit
	logBuffer      []string
	showLogs       bool
	connected      bool
	companion      *service.Companion
}

func Run(companion *service.Companion, cfg Config) {
	runtime.LockOSThread()

	if cfg.MQTTUrl == "" {
		cfg.MQTTUrl = "tcp://localhost:1883"
	}
	if cfg.RLAddress == "" {
		cfg.RLAddress = "ws://127.0.0.1:49124"
	}

	w := &MainWindow{companion: companion}
	w.create(cfg)
	w.events()

	companion.SetStatusCallback(func(msg string) {
		w.wnd.UiThread(func() {
			w.appendLog(msg)
		})
	})

	companion.SetConnectionStateCallback(func(mqttOk, rlOk bool) {
		w.wnd.UiThread(func() {
			w.updateStatus(mqttOk, rlOk)
		})
	})

	autoConnected := false
	bannerLoaded := false
	w.wnd.On().WmShowWindow(func(p ui.WmShowWindow) {
		if !bannerLoaded {
			bannerLoaded = true
			w.loadBanner()
		}
		if !autoConnected && w.chkAutoConnect.IsChecked() {
			autoConnected = true
			w.connect()
		}
	})

	w.wnd.RunAsMain()
}

func (w *MainWindow) create(cfg Config) {
	w.wnd = ui.NewMain(
		ui.OptsMain().
			Title("rlapi2mqtt").
			Size(ui.Dpi(460, compactHeight)),
	)

	// The banner is alpha-blended onto the window background. A plain
	// SS_BITMAP static control cannot respect the 32bpp alpha channel (it
	// renders transparent pixels as black), so we paint it ourselves in
	// WM_PAINT. The window class fills the background (dialog gray), so we
	// only blend a fully-composited logo-rect on top.
	w.wnd.On().WmPaint(func() {
		w.paintBanner()
	})

	y := bannerHeight // vertical offset for all controls below the banner

	w.lblBroker = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Mosquitto Broker URL").
			Position(ui.Dpi(10, y+12)),
	)

	w.txtBroker = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTUrl).
			Position(ui.Dpi(10, y+32)).
			Width(ui.DpiX(320)),
	)

	w.lblUser = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Username").
			Position(ui.Dpi(10, y+62)),
	)

	w.txtUser = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTUsername).
			Position(ui.Dpi(10, y+82)).
			Width(ui.DpiX(150)),
	)

	w.lblPass = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Password").
			Position(ui.Dpi(170, y+62)),
	)

	w.txtPass = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTPassword).
			Position(ui.Dpi(170, y+82)).
			Width(ui.DpiX(150)).
			CtrlStyle(co.ES_PASSWORD),
	)

	w.lblGameWS = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Rocket League Stats API Address").
			Position(ui.Dpi(10, y+112)),
	)

	w.txtGameWS = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.RLAddress).
			Position(ui.Dpi(10, y+132)).
			Width(ui.DpiX(320)),
	)

	w.btnConnect = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Connect").
			Position(ui.Dpi(340, y+32)).
			Width(ui.DpiX(100)),
	)

	w.btnSaveConfig = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Save Config").
			Position(ui.Dpi(340, y+62)).
			Width(ui.DpiX(100)),
	)

	w.lblMQTTStatus = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("MQTT: --").
			Position(ui.Dpi(340, y+95)).
			Size(ui.Dpi(110, 15)),
	)

	w.lblRLStatus = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("RL: --").
			Position(ui.Dpi(340, y+112)).
			Size(ui.Dpi(110, 15)),
	)

	chkAutoConnectOpts := ui.OptsCheckBox().
		Text("Auto connect").
		Position(ui.Dpi(290, y+165))
	if cfg.AutoConnect {
		chkAutoConnectOpts.State(co.BST_CHECKED)
	}
	w.chkAutoConnect = ui.NewCheckBox(w.wnd, chkAutoConnectOpts)

	w.chkShowLogs = ui.NewCheckBox(
		w.wnd,
		ui.OptsCheckBox().
			Text("Show logs").
			Position(ui.Dpi(10, y+165)),
	)

	w.chkLimitUpdate = ui.NewCheckBox(
		w.wnd,
		ui.OptsCheckBox().
			Text("Limit UpdateState 1/s").
			Position(ui.Dpi(120, y+165)).
			State(co.BST_CHECKED),
	)

	w.lblLog = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Log").
			Position(ui.Dpi(10, y+185)).
			WndStyle(co.WS_CHILD|co.WS_GROUP),
	)

	w.txtLog = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Position(ui.Dpi(10, y+205)).
			Width(ui.DpiX(435)).
			Height(ui.DpiY(190)).
			CtrlStyle(co.ES_MULTILINE|co.ES_AUTOVSCROLL|co.ES_READONLY|co.ES_NOHIDESEL).
			WndStyle(co.WS_CHILD|co.WS_TABSTOP|co.WS_GROUP|co.WS_VSCROLL),
	)
}

func (w *MainWindow) appendLog(msg string) {
	w.logBuffer = append(w.logBuffer, msg)
	if len(w.logBuffer) > 100 {
		w.logBuffer = w.logBuffer[len(w.logBuffer)-100:]
	}

	if w.showLogs {
		w.flushLog()
	}
}

func (w *MainWindow) flushLog() {
	newText := strings.Join(w.logBuffer, "\r\n")
	w.txtLog.SetText(newText)
	end := len(newText)
	w.txtLog.SetSelection(end, end)
	w.txtLog.Hwnd().SendMessage(co.EM_SCROLLCARET, 0, win.LPARAM(0))
}

func (w *MainWindow) setLogVisible(visible bool) {
	w.showLogs = visible
	if visible {
		w.resizeWindow(expandedHeight)
		w.lblLog.Hwnd().ShowWindow(co.SW_SHOWNORMAL)
		w.txtLog.Hwnd().ShowWindow(co.SW_SHOWNORMAL)
		w.flushLog()
	} else {
		w.lblLog.Hwnd().ShowWindow(co.SW_HIDE)
		w.txtLog.Hwnd().ShowWindow(co.SW_HIDE)
		w.resizeWindow(compactHeight)
	}
}

func (w *MainWindow) resizeWindow(clientHeight int) {
	hwnd := w.wnd.Hwnd()
	windowRect, _ := hwnd.GetWindowRect()
	clientRect, _ := hwnd.GetClientRect()

	chromeWidth := (windowRect.Right - windowRect.Left) - (clientRect.Right - clientRect.Left)
	chromeHeight := (windowRect.Bottom - windowRect.Top) - (clientRect.Bottom - clientRect.Top)

	newW := int32(ui.DpiX(460)) + chromeWidth
	newH := int32(ui.DpiY(clientHeight)) + chromeHeight

	hwnd.SetWindowPos(0,
		win.POINT{},
		win.SIZE{Cx: newW, Cy: newH},
		co.SWP_NOMOVE|co.SWP_NOZORDER|co.SWP_NOACTIVATE,
	)
}

func (w *MainWindow) updateStatus(mqttOk, rlOk bool) {
	if mqttOk {
		w.lblMQTTStatus.Hwnd().SetWindowText("MQTT: Connected")
	} else {
		w.lblMQTTStatus.Hwnd().SetWindowText("MQTT: Disconnected")
	}
	if rlOk {
		w.lblRLStatus.Hwnd().SetWindowText("RL: Connected")
	} else {
		w.lblRLStatus.Hwnd().SetWindowText("RL: Disconnected")
	}
}

func (w *MainWindow) connect() {
	brokerURL := w.txtBroker.Text()
	if brokerURL == "" {
		brokerURL = "tcp://localhost:1883"
	}
	gameAddr := w.txtGameWS.Text()
	if gameAddr == "" {
		gameAddr = "ws://127.0.0.1:49124"
	}

	username := strings.TrimSpace(w.txtUser.Text())
	password := strings.TrimSpace(w.txtPass.Text())

	w.appendLog("Starting connection loop...")
	w.companion.StartLoop(brokerURL, username, password, gameAddr)
	w.connected = true
	w.btnConnect.Hwnd().SetWindowText("Disconnect")
}

func (w *MainWindow) disconnect() {
	w.companion.StopLoop()
	w.connected = false
	w.btnConnect.Hwnd().SetWindowText("Connect")
}

func (w *MainWindow) saveConfig() {
	autoConnect := "false"
	if w.chkAutoConnect.IsChecked() {
		autoConnect = "true"
	}

	content := fmt.Sprintf("MQTT_URL=%s\nMQTT_USERNAME=%s\nMQTT_PASSWORD=%s\nRL_ADDRESS=%s\nAUTO_CONNECT=%s\n",
		strings.TrimSpace(w.txtBroker.Text()),
		strings.TrimSpace(w.txtUser.Text()),
		strings.TrimSpace(w.txtPass.Text()),
		strings.TrimSpace(w.txtGameWS.Text()),
		autoConnect,
	)

	err := os.WriteFile("rlapi2mqtt.ini", []byte(content), 0644)
	if err != nil {
		w.appendLog("Failed to save config: " + err.Error())
	} else {
		w.appendLog("Config saved to rlapi2mqtt.ini")
	}
}

func (w *MainWindow) events() {
	w.btnConnect.On().BnClicked(func() {
		if w.connected {
			w.disconnect()
		} else {
			w.connect()
		}
	})

	w.btnSaveConfig.On().BnClicked(func() {
		w.saveConfig()
	})

	w.chkShowLogs.On().BnClicked(func() {
		w.setLogVisible(w.chkShowLogs.IsChecked())
	})

	w.companion.SetLimitUpdateState(true)
	w.chkLimitUpdate.On().BnClicked(func() {
		w.companion.SetLimitUpdateState(w.chkLimitUpdate.IsChecked())
	})
}

func (w *MainWindow) loadBanner() {
	// Guard against missing/broken GDI procedures on unusual systems: a
	// cosmetic banner must never crash the app, so any failure here simply
	// leaves the banner undrawn.
	defer func() {
		if r := recover(); r != nil {
			w.bannerDc = 0
			w.bannerBmp = 0
		}
	}()

	data := assets.BannerBMP
	if len(data) < 54 {
		return
	}

	// BMP file header: pixel data offset at bytes 10-13.
	pixelOffset := binary.LittleEndian.Uint32(data[10:14])

	// DIB header fields (BITMAPV5HEADER here): width/height at 18/22, bpp at 28.
	srcW := int(binary.LittleEndian.Uint32(data[18:22]))
	srcH := int32(binary.LittleEndian.Uint32(data[22:26]))
	if srcH < 0 {
		srcH = -srcH // negative height = top-down DIB
	}
	bpp := int(binary.LittleEndian.Uint16(data[28:30]))
	if bpp != 32 || srcW != bannerImageW || int(srcH) != bannerImageH {
		return
	}
	if int(pixelOffset)+srcW*int(srcH)*4 > len(data) {
		return
	}

	// Build our own 32bpp DIB (dimensions from the constants), fully opaque.
	var bih win.BITMAPINFOHEADER
	bih.SetBiSize()
	bih.Width = bannerImageW
	bih.Height = -bannerImageH // top-down
	bih.Planes = 1
	bih.BitCount = 32
	bih.Compression = co.BI_RGB
	bmi := win.BITMAPINFO{BmiHeader: bih}

	hBmp, pBits, err := win.HDC(0).CreateDIBSection(
		&bmi,
		co.DIB_COLORS_RGB,
		win.HFILEMAP(0),
		0,
	)
	if err != nil || hBmp == 0 || pBits == nil {
		return
	}

	// Background color of the dialog (COLOR_BTNFACE). Composite the logo onto
	// this in software so the result is fully opaque and needs only a plain
	// BitBlt to draw — no alpha blending required.
	bg := win.GetSysColor(co.COLOR_BTNFACE)
	bgB := byte(bg & 0xff)
	bgG := byte((bg >> 8) & 0xff)
	bgR := byte((bg >> 16) & 0xff)

	n := srcW * int(srcH) * 4
	px := unsafe.Slice(pBits, n)
	src := data[pixelOffset:]
	for i := 0; i < n; i += 4 {
		sb, sg, sr, a := src[i], src[i+1], src[i+2], src[i+3]
		if a >= 250 { // effectively opaque: use source color
			px[i], px[i+1], px[i+2], px[i+3] = sb, sg, sr, 255
			continue
		}
		if a == 0 { // fully transparent: background
			px[i], px[i+1], px[i+2], px[i+3] = bgB, bgG, bgR, 255
			continue
		}
		// Partial alpha: blend source over background.
		px[i] = uint8((uint16(sb)*uint16(a) + uint16(bgB)*uint16(255-a)) / 255)
		px[i+1] = uint8((uint16(sg)*uint16(a) + uint16(bgG)*uint16(255-a)) / 255)
		px[i+2] = uint8((uint16(sr)*uint16(a) + uint16(bgR)*uint16(255-a)) / 255)
		px[i+3] = 255
	}

	// Select the DIB into a memory DC so it can be used as a BitBlt source.
	hdcMem, err := win.HDC(0).CreateCompatibleDC()
	if err != nil {
		return
	}
	if _, err := hdcMem.SelectObjectBmp(hBmp); err != nil {
		hdcMem.DeleteDC()
		return
	}

	w.bannerDc = hdcMem
	w.bannerBmp = hBmp

	// Trigger a repaint so the banner is drawn immediately.
	_ = w.wnd.Hwnd().InvalidateRect(nil, true)
}

// paintBanner is the WM_PAINT handler. It begins the paint cycle (which also
// lets the window class fill the background) and draws the composited logo
// rect on top. The drawing is wrapped so that if a GDI procedure is
// unavailable on the system (e.g. an unavailable drawing procedure), we
// degrade to no banner instead of crashing.
func (w *MainWindow) paintBanner() {
	if w.bannerDc == 0 {
		return
	}
	var ps win.PAINTSTRUCT
	hdc, err := w.wnd.Hwnd().BeginPaint(&ps)
	if err != nil {
		return
	}
	defer w.wnd.Hwnd().EndPaint(&ps)
	defer func() {
		if r := recover(); r != nil {
			w.bannerDc = 0
			w.bannerBmp = 0
		}
	}()
	w.drawBanner(hdc)
}

// drawBanner copies the logo rect from the pre-composited memory DC onto the
// given HDC with a plain BitBlt (the pixels are already fully opaque, so no
// alpha blending is needed). source DIB.
func (w *MainWindow) drawBanner(hdc win.HDC) {
	if w.bannerDc == 0 {
		return
	}
	_ = hdc.BitBlt(
		win.POINT{X: int32(ui.DpiX(bannerLeft)), Y: int32(ui.DpiY(bannerTop))},
		win.SIZE{Cx: int32(ui.DpiX(bannerImageW)), Cy: int32(ui.DpiY(bannerImageH))},
		w.bannerDc,
		win.POINT{},
		co.ROP_SRCCOPY,
	)
}
