package gui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"

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
	compactHeight  = 210
	expandedHeight = 430
)

type MainWindow struct {
	wnd            *ui.Main
	lblBroker      *ui.Static
	txtBroker      *ui.Edit
	lblUser        *ui.Static
	txtUser        *ui.Edit
	lblPass        *ui.Static
	txtPass        *ui.Edit
	lblGameWS      *ui.Static
	txtGameWS      *ui.Edit
	btnConnect     *ui.Button
	btnDisconnect  *ui.Button
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
	companion      *service.Companion
}

func Run(companion *service.Companion, cfg Config) {
	runtime.LockOSThread()

	if cfg.MQTTUrl == "" {
		cfg.MQTTUrl = "tcp://localhost:1883"
	}
	if cfg.RLAddress == "" {
		cfg.RLAddress = "127.0.0.1:49123"
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
	w.wnd.On().WmShowWindow(func(p ui.WmShowWindow) {
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

	w.lblBroker = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Mosquitto Broker URL").
			Position(ui.Dpi(10, 12)),
	)

	w.txtBroker = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTUrl).
			Position(ui.Dpi(10, 32)).
			Width(ui.DpiX(320)),
	)

	w.lblUser = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Username").
			Position(ui.Dpi(10, 62)),
	)

	w.txtUser = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTUsername).
			Position(ui.Dpi(10, 82)).
			Width(ui.DpiX(150)),
	)

	w.lblPass = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Password").
			Position(ui.Dpi(170, 62)),
	)

	w.txtPass = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.MQTTPassword).
			Position(ui.Dpi(170, 82)).
			Width(ui.DpiX(150)).
			CtrlStyle(co.ES_PASSWORD),
	)

	w.lblGameWS = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Rocket League Stats API Address").
			Position(ui.Dpi(10, 112)),
	)

	w.txtGameWS = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Text(cfg.RLAddress).
			Position(ui.Dpi(10, 132)).
			Width(ui.DpiX(320)),
	)

	w.btnConnect = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Connect").
			Position(ui.Dpi(340, 32)).
			Width(ui.DpiX(100)),
	)

	w.btnDisconnect = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Disconnect").
			Position(ui.Dpi(340, 62)).
			Width(ui.DpiX(100)),
	)

	w.btnSaveConfig = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Save Config").
			Position(ui.Dpi(340, 132)).
			Width(ui.DpiX(100)),
	)

	w.lblMQTTStatus = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("MQTT: --").
			Position(ui.Dpi(340, 95)).
			Size(ui.Dpi(110, 15)),
	)

	w.lblRLStatus = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("RL: --").
			Position(ui.Dpi(340, 112)).
			Size(ui.Dpi(110, 15)),
	)

	chkAutoConnectOpts := ui.OptsCheckBox().
		Text("Auto connect").
		Position(ui.Dpi(290, 165))
	if cfg.AutoConnect {
		chkAutoConnectOpts.State(co.BST_CHECKED)
	}
	w.chkAutoConnect = ui.NewCheckBox(w.wnd, chkAutoConnectOpts)

	w.chkShowLogs = ui.NewCheckBox(
		w.wnd,
		ui.OptsCheckBox().
			Text("Show logs").
			Position(ui.Dpi(10, 165)),
	)

	w.chkLimitUpdate = ui.NewCheckBox(
		w.wnd,
		ui.OptsCheckBox().
			Text("Limit UpdateState 1/s").
			Position(ui.Dpi(120, 165)).
			State(co.BST_CHECKED),
	)

	w.lblLog = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Log").
			Position(ui.Dpi(10, 185)).
			WndStyle(co.WS_CHILD|co.WS_GROUP),
	)

	w.txtLog = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Position(ui.Dpi(10, 205)).
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
		w.lblMQTTStatus.SetTextAndResize("MQTT: Connected")
	} else {
		w.lblMQTTStatus.SetTextAndResize("MQTT: Disconnected")
	}
	if rlOk {
		w.lblRLStatus.SetTextAndResize("RL: Connected")
	} else {
		w.lblRLStatus.SetTextAndResize("RL: Disconnected")
	}
	w.wnd.Hwnd().InvalidateRect(nil, true)
}

func (w *MainWindow) connect() {
	brokerURL := w.txtBroker.Text()
	if brokerURL == "" {
		brokerURL = "tcp://localhost:1883"
	}
	gameAddr := w.txtGameWS.Text()
	if gameAddr == "" {
		gameAddr = "127.0.0.1:49123"
	}

	username := strings.TrimSpace(w.txtUser.Text())
	password := strings.TrimSpace(w.txtPass.Text())

	w.appendLog("Starting connection loop...")
	w.companion.StartLoop(brokerURL, username, password, gameAddr)
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
		w.connect()
	})

	w.btnDisconnect.On().BnClicked(func() {
		w.companion.StopLoop()
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
