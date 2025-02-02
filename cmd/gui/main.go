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
	"github.com/agukrapo/playlist-creator/internal/logs"
	"github.com/agukrapo/playlist-creator/internal/random"
	"github.com/agukrapo/playlist-creator/internal/results"
	"github.com/agukrapo/playlist-creator/playlists"
)

const appTitle = "playlist-creator"

func main() {
	cookie, err := env.Lookup[string]("DEEZER_ARL_COOKIE")
	if err != nil {
		fyne.LogError("env.Lookup", err)
	}

	logFile, err := logs.NewFile(appTitle)
	if err != nil {
		fyne.LogError("logs.NewFile", err)
		os.Exit(1)
	}
	defer logFile.Close()

	app := newApplication(cookie, logs.New(logFile))
	app.ShowAndRun()
}

type application struct {
	window fyne.Window

	formA   *widget.Form
	results []*playlists.Track

	dialogs chan dialoger

	cookie string

	log *logs.Logger
}

func newApplication(cookie string, log *logs.Logger) *application {
	out := fyneapp.New()

	version := out.Metadata().Custom["version"]
	if version == "" {
		version = "dev"
	}

	w := out.NewWindow(fmt.Sprintf("%s %s", appTitle, version))
	w.Resize(fyne.NewSize(1300, 800))

	return &application{
		window:  w,
		dialogs: make(chan dialoger),
		cookie:  cookie,
		log:     log,
	}
}

func (a *application) ShowAndRun() {
	go a.dialogsLoop()

	a.renderNewFormA()
	a.window.ShowAndRun()
}

func (a *application) renderNewFormA() {
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
			a.error(err)
			return
		}

		a.working()

		target := deezer.New(client.New(), arl.Text, a.log)
		a.renderResults(target, name.Text, splitLines(songs.Text))
	}

	form.Append("ARL", arl)
	form.Append("Name", name)
	form.Append("Songs", songs)

	a.window.SetContent(page("Playlist data", form))
	a.formA = form
	a.results = nil
	reset()
}

func (a *application) renderResults(target playlists.Target, name string, songs []string) {
	if len(a.results) != len(songs) {
		size := len(songs)
		if len(a.results) < len(songs) {
			size = len(a.results)
		}

		tmp := make([]*playlists.Track, len(songs))
		for i := 0; i < size; i++ {
			tmp[i] = a.results[i]
		}
		a.results = tmp
	}

	items := make([]*widget.FormItem, 0, len(songs))
	for i, song := range songs {
		items = append(items, &widget.FormItem{
			Text:   fmt.Sprintf("%d. %s", i+1, strings.TrimPrefix(song, playlists.Locked)),
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
			entry, ok := a.formA.Items[2].Widget.(*widget.Entry)
			if !ok {
				panic("not an entry, should ever happen")
			}

			lines := splitLines(entry.Text)
			for i, line := range lines {

				line = strings.TrimPrefix(line, playlists.Locked)
				if a.results[i] != nil {
					line = playlists.Locked + line
				}

				lines[i] = line
			}

			entry.Text = strings.Join(lines, "\n")
			a.formA.Items[2].Widget = entry

			a.window.SetContent(page("Playlist data", a.formA))
		},
		OnSubmit: func() {
			if !data.Empty() {
				cnf := a.makeConfirm(manager, name, data)
				cnf.Show()
			}
		},
	}

	if err := manager.Gather(context.Background(), songs, func(i int, _ string, matches []playlists.Track) {
		singleResult := func(track *playlists.Track, checked bool) {
			check := widget.NewCheck("", func(v bool) {
				if v {
					a.results[i] = track
				} else {
					a.results[i] = nil
				}
			})
			check.Checked = checked

			var w fyne.CanvasObject = container.NewHBox(check, widget.NewLabel(track.Name))
			if ok, addedAt := data.Add(i, track.ID); !ok {
				w = errorLabel(i+1, fmt.Sprintf("duplicated of track %d %q", addedAt+1, track.Name))
			}

			items[i].Widget = w
		}

		if i < len(a.results) && a.results[i] != nil {
			singleResult(a.results[i], true)
			return
		}

		if len(matches) == 0 {
			items[i].Widget = errorLabel(i+1, "not found")
			return
		}

		if len(matches) == 1 {
			singleResult(&matches[0], false)
			return
		}

		opts := make([]string, 0, len(matches))
		for _, t := range matches {
			opts = append(opts, t.Name)
		}

		sel := widget.NewSelect(opts, nil)

		check := widget.NewCheck("", func(v bool) {
			if v {
				a.results[i] = &matches[sel.SelectedIndex()]
			} else {
				a.results[i] = nil
			}
		})

		sel.OnChanged = func(_ string) {
			track := matches[sel.SelectedIndex()]
			if ok, addedAt := data.Add(i, track.ID); !ok {
				a.notify(fmt.Sprintf("track %d: duplicated of track %d %q", i+1, addedAt+1, track.Name))
			}
			check.Checked = false
			a.results[i] = nil
		}
		sel.SetSelectedIndex(0)

		items[i].Widget = container.NewBorder(nil, nil, check, nil, sel)
	}); err != nil {
		a.error(err)
		return
	}

	a.window.SetContent(page("Search results", container.NewVScroll(form)))
	a.renderDialog(nothing{})
}

func (a *application) makeConfirm(manager *playlists.Manager, name string, data *results.Set) *dialog.ConfirmDialog {
	songs := data.Slice()
	return dialog.NewConfirm("Create playlist?", fmt.Sprintf("name %q\n%d tracks", name, len(songs)), func(b bool) {
		if !b {
			return
		}
		a.working()

		if err := manager.Push(context.Background(), name, songs); err != nil {
			a.error(err)
			return
		}

		a.renderNewFormA()
		a.renderDialog(nothing{})
	}, a.window)
}

func errorLabel(trackNumber int, msg string) fyne.CanvasObject {
	_, _ = fmt.Fprintf(os.Stderr, "Error: track %d: %s\n", trackNumber, msg)
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

func splitLines(in string) []string {
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
