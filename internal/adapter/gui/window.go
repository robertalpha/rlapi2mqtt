package gui

import (
	"runtime"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"

	"rla-companion/internal/port"
)

type MainWindow struct {
	wnd       *ui.Main
	lblBroker *ui.Static
	txtBroker *ui.Edit
	btnConnect *ui.Button
	lblResult *ui.Static
	txtResult *ui.Edit
	broker    port.BrokerClient
}

func Run(broker port.BrokerClient) {
	runtime.LockOSThread()

	w := &MainWindow{broker: broker}
	w.create()
	w.events()
	w.wnd.RunAsMain()
}

func (w *MainWindow) create() {
	w.wnd = ui.NewMain(
		ui.OptsMain().
			Title("RLA Companion").
			Size(ui.Dpi(420, 220)),
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
			Text("tcp://localhost:1883").
			Position(ui.Dpi(10, 32)).
			Width(ui.DpiX(300)),
	)

	w.btnConnect = ui.NewButton(
		w.wnd,
		ui.OptsButton().
			Text("&Connect").
			Position(ui.Dpi(320, 30)),
	)

	w.lblResult = ui.NewStatic(
		w.wnd,
		ui.OptsStatic().
			Text("Connection Result").
			Position(ui.Dpi(10, 65)),
	)

	w.txtResult = ui.NewEdit(
		w.wnd,
		ui.OptsEdit().
			Position(ui.Dpi(10, 85)).
			Width(ui.DpiX(395)).
			Height(ui.DpiY(100)).
			CtrlStyle(co.ES_MULTILINE|co.ES_AUTOVSCROLL|co.ES_READONLY|co.ES_NOHIDESEL),
	)
}

func (w *MainWindow) events() {
	w.btnConnect.On().BnClicked(func() {
		brokerURL := w.txtBroker.Text()
		if brokerURL == "" {
			brokerURL = "tcp://localhost:1883"
		}

		w.txtResult.SetText("Connecting to " + brokerURL + "...")

		err := w.broker.Connect(brokerURL)
		if err != nil {
			w.txtResult.SetText("Connection failed:\r\n" + err.Error())
			return
		}

		w.txtResult.SetText("Successfully connected to " + brokerURL)
	})
}
