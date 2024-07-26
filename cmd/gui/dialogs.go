package main

import (
	"fmt"
	"image/color"
	"os"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type dialoger interface {
	Show()
	Hide()
}

func (a *application) renderDialog(dialog dialoger) {
	select {
	case a.dialogs <- dialog:
	default:
	}
}

func (a *application) dialogsLoop() {
	var prev dialoger

	for d := range a.dialogs {
		if prev != nil {
			prev.Hide()
		}
		prev = d
		d.Show()
	}
}

type modal struct {
	window   fyne.Window
	dialog   *dialog.CustomDialog
	activity *widget.Activity
	on       bool
	mu       sync.Mutex
}

func (a *application) notify(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, "Error:", msg)

	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   appTitle,
		Content: msg,
	})
}

func (a *application) error(msg any) {
	_, _ = fmt.Fprintln(os.Stderr, "Error:", msg)
	a.renderDialog(dialog.NewError(fmt.Errorf("%v", msg), a.window))
}

func (a *application) working() {
	a.renderDialog(&modal{
		window: a.window,
	})
}

func (m *modal) Show() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.on {
		return
	}

	m.on = true

	prop := canvas.NewRectangle(color.Transparent)
	prop.SetMinSize(fyne.NewSize(50, 50))

	m.activity = widget.NewActivity()
	m.dialog = dialog.NewCustomWithoutButtons("Please wait...", container.NewStack(prop, m.activity), m.window)
	m.activity.Start()
	m.dialog.Show()
}

func (m *modal) Hide() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.on {
		return
	}

	m.on = false

	m.activity.Stop()
	m.dialog.Hide()
}

type nothing struct{}

func (nothing) Show() {}
func (nothing) Hide() {}
