package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/agukrapo/go-http-client/client"
	"github.com/agukrapo/playlist-creator/deezer"
	"github.com/agukrapo/playlist-creator/internal/env"
	"github.com/agukrapo/playlist-creator/internal/random"
	"github.com/agukrapo/playlist-creator/internal/results"
	"github.com/agukrapo/playlist-creator/playlists"
)

func main() {
	cookie, err := env.Lookup[string]("DEEZER_ARL_COOKIE")
	if err != nil {
		fyne.LogError("env.Lookup", err)
	}

	app := newApplication(cookie)
	app.ShowAndRun()
}

type application struct {
	window fyne.Window
	form   *widget.Form

	dialogs chan dialoger

	cookie string
}

func newApplication(cookie string) *application {
	out := fyneapp.New()
	w := out.NewWindow("Playlist Creator")
	w.Resize(fyne.NewSize(1300, 800))

	return &application{
		window:  w,
		dialogs: make(chan dialoger),
		cookie:  cookie,
	}
}

func (a *application) ShowAndRun() {
	go a.dialogsLoop()

	a.renderForm()
	a.window.ShowAndRun()
}

func (a *application) renderForm() {
	arl := widget.NewEntry()
	arl.Validator = notEmpty("ARL")

	name := widget.NewEntry()
	name.Validator = notEmpty("name")

	songs := widget.NewMultiLineEntry()
	songs.SetMinRowsVisible(30)
	songs.Validator = notEmpty("songs")
	songs.OnChanged = func(_ string) {
		_ = songs.Validate() // force submit button to enable after a paste
	}

	reset := func() {
		arl.SetText(a.cookie)
		name.SetText("NAME " + random.Name(20))
		songs.SetText("")
	}

	form := &widget.Form{
		SubmitText: "Search tracks",
		CancelText: "Reset",
		OnCancel:   reset,
	}

	form.OnSubmit = func() {
		if err := form.Validate(); err != nil {
			a.notify(err)
			return
		}

		a.showModal()

		target := deezer.New(client.New(), arl.Text)
		a.renderResults(target, name.Text, lines(songs.Text))
	}

	form.Append("ARL", arl)
	form.Append("Name", name)
	form.Append("Songs", songs)

	a.window.SetContent(page("Playlist data", form))
	a.form = form
	reset()
}

func (a *application) renderResults(target playlists.Target, name string, songs []string) {
	items := make([]*widget.FormItem, 0, len(songs))
	for _, song := range songs {
		items = append(items, &widget.FormItem{
			Text:   song,
			Widget: widget.NewLabel("Searching..."),
		})
	}

	data := results.New(len(songs))
	manager := playlists.NewManager(target, 100)

	form := &widget.Form{
		Items:      items,
		SubmitText: "Create playlist",
		CancelText: "Back",
		OnCancel: func() {
			a.window.SetContent(page("Playlist data", a.form))
		},
		OnSubmit: func() {
			if data.Length() > 0 {
				cnf := a.makeConfirm(manager, name, data)
				cnf.Show()
			}
		},
	}

	if err := manager.Gather(context.Background(), songs, func(i int, query string, matches []playlists.Track) {
		if len(matches) == 0 {
			items[i].Widget = errorLabel(query, "Not found")
			return
		}

		if len(matches) == 1 {
			track := matches[0]
			var w fyne.CanvasObject = widget.NewLabel(track.Name)
			if !data.Add(i, track.ID, track.Name) {
				w = errorLabel(query, fmt.Sprintf("Duplicated result: id %s, name %q", track.ID, track.Name))
			}
			items[i].Widget = w
			return
		}

		opts := make([]string, 0, len(matches))
		for _, t := range matches {
			opts = append(opts, t.Name)
		}

		s := widget.NewSelect(opts, nil)
		s.OnChanged = func(_ string) {
			track := matches[s.SelectedIndex()]
			if !data.Add(i, track.ID, track.Name) {
				a.notify(fmt.Sprintf("Duplicated track: id %s, Name %q", track.ID, track.Name))
			}
		}
		s.SetSelectedIndex(0)

		items[i].Widget = s
	}); err != nil {
		a.notify(err)
		return
	}

	a.window.SetContent(page("Search results", container.NewVScroll(form)))
}

func (a *application) makeConfirm(manager *playlists.Manager, name string, data *results.Set) *dialog.ConfirmDialog {
	return dialog.NewConfirm("Create playlist?", fmt.Sprintf("name %q\n%d tracks", name, data.Length()), func(b bool) {
		if !b {
			return
		}
		a.showModal()

		if err := manager.Push(context.Background(), name, data.Slice()); err != nil {
			a.notify(err)
			return
		}
		a.renderForm()

		a.renderDialog(nothing{})
	}, a.window)
}

func errorLabel(query, msg string) fyne.CanvasObject {
	_, _ = fmt.Fprintf(os.Stderr, "Error: %q: %s\n", query, msg)
	return container.NewHBox(widget.NewIcon(theme.ErrorIcon()),
		widget.NewLabelWithStyle(msg, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Italic: true}))
}

func notEmpty(name string) func(v string) error {
	return func(v string) error {
		if v == "" {
			return errors.New("empty " + name)
		}
		return nil
	}
}

func lines(in string) []string {
	var out []string
	for _, line := range strings.Split(in, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func page(title string, content fyne.CanvasObject) fyne.CanvasObject {
	makeCell := func() fyne.CanvasObject {
		rect := canvas.NewRectangle(nil)
		rect.SetMinSize(fyne.NewSize(10, 10))
		return rect
	}

	label := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewBorder(container.NewVBox(label, widget.NewSeparator(), makeCell()), makeCell(), makeCell(), makeCell(), content)
}
