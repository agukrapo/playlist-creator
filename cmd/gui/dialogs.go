package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/agukrapo/playlist-creator/internal/results"
	"github.com/agukrapo/playlist-creator/playlists"
)

type dialoger interface {
	Show()
	Hide()
}

func (a *application) renderDialog(dialog dialoger) {
	select {
	case a.dialogs <- dialog:
	case <-time.After(500 * time.Millisecond):
		a.error("dialog timeout")
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

func (a *application) confirmPlaylist(manager *playlists.Manager, name string, data *results.Set) *dialog.FormDialog {
	songs, excluded := data.Slice()

	nw := widget.NewEntry()
	nw.Validator = notEmpty("Name")
	nw.SetText(name)

	ew := widget.NewMultiLineEntry()
	ew.SetMinRowsVisible(10)
	ew.Text = strings.Join(excluded, "\n")

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nw),
		widget.NewFormItem("Tracks", widget.NewLabel(strconv.Itoa(len(songs)))),
		widget.NewFormItem("Excluded", ew),
	}

	out := dialog.NewForm("Create playlist?", "Create", "Cancel", items, func(b bool) {
		if !b {
			return
		}

		a.working()

		if playlistID, err := manager.Push(context.Background(), nw.Text, songs); err != nil {
			a.error(err)
		} else {
			a.renderDialog(a.playlistInfo(playlistID))
		}
	}, a.window)

	out.Resize(fyne.NewSize(600, 400))

	return out
}

func (a *application) playlistInfo(playlistID string) *dialog.CustomDialog {
	url := "https://www.deezer.com/us/playlist/" + playlistID // TODO make target agnostic

	bottom := container.NewHBox(
		layout.NewSpacer(),
		widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
			a.window.Clipboard().SetContent(url)
		}),
	)

	content := container.NewBorder(nil, nil, nil, bottom, widget.NewLabel(url))

	out := dialog.NewCustom("Playlist created", "Dismiss", content, a.window)

	out.SetOnClosed(func() {
		a.renderNewFormA()
		a.renderDialog(nothing{})
	})

	return out
}

type nothing struct{}

func (nothing) Show() {}
func (nothing) Hide() {}
