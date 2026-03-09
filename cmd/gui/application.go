package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
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
	"github.com/agukrapo/playlist-creator/internal/logs"
	"github.com/agukrapo/playlist-creator/internal/random"
	"github.com/agukrapo/playlist-creator/internal/results"
	"github.com/agukrapo/playlist-creator/playlists"
)

type application struct {
	window fyne.Window

	formA *widget.Form

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
	arl.SetText(a.cookie)

	name := widget.NewEntry()
	name.Validator = notEmpty("name")
	name.SetText("NAME " + random.Name(20))

	songs := widget.NewMultiLineEntry()
	songs.SetMinRowsVisible(30)
	songs.Validator = notEmpty("songs")
	songs.OnChanged = func(_ string) {
		_ = songs.Validate() // force submit button to enable after a paste
	}

	reset := func() {
		a.renderNewFormA()
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
}

func (a *application) renderResults(target playlists.Target, name string, songs []results.Item) {
	items := make([]*widget.FormItem, 0, len(songs))
	for i, song := range songs {
		items = append(items, &widget.FormItem{
			Text:   fmt.Sprintf("%d. %s", i+1, song.Query()),
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
				panic("not an form entry, should never happen")
			}

			entry.Text = strings.Join(data.Queries(), "\n")
			a.formA.Items[2].Widget = entry
			a.formA.Refresh()

			a.window.SetContent(page("Playlist data", a.formA))
		},
		OnSubmit: func() {
			if !data.Empty() {
				cnf := a.makeConfirm(manager, name, data)
				cnf.Show()
			}
		},
	}

	if err := manager.Gather(context.Background(), songs, func(i int, item results.Item, matches []playlists.Track) {
		singleResult := func(item results.Item) {
			check := widget.NewCheck("", nil)
			check.OnChanged = func(v bool) {
				item := item.WithActive(v)
				if ok, addedAt := data.Put(i, item); !ok {
					a.notify(fmt.Sprintf("track %d: duplicated of track %d %q", i+1, addedAt+1, item.Name()))
					check.Checked = false
					check.Disable()
					return
				}

				check.Checked = item.Active()
			}

			check.OnChanged(item.Active())

			items[i].Widget = container.NewHBox(check, widget.NewLabel(item.Name()))
		}

		if item.Active() {
			singleResult(item)
			return
		}

		if len(matches) == 0 {
			items[i].Widget = errorLabel(i+1, "not found")
			if ok, addedAt := data.Put(i, item); !ok {
				a.notify(fmt.Sprintf("track %d: duplicated of track %d %q", i+1, addedAt+1, item.Name()))
			}
			return
		}

		if len(matches) == 1 {
			singleResult(item.WithID(matches[0].ID).WithName(matches[0].Name))
			return
		}

		opts := make([]string, 0, len(matches))
		for _, t := range matches {
			opts = append(opts, t.Name)
		}

		sel := widget.NewSelect(opts, nil)
		sel.SetSelectedIndex(0)

		check := widget.NewCheck("", nil)
		check.OnChanged = func(v bool) {
			if v {
				sel.Disable()
			} else {
				sel.Enable()
			}

			track := matches[sel.SelectedIndex()]
			if ok, addedAt := data.Put(i, item.WithID(track.ID).WithName(track.Name).WithActive(v)); !ok {
				a.notify(fmt.Sprintf("track %d: duplicated of track %d %q", i+1, addedAt+1, track.Name))
				check.Checked = false
				check.Disable()
				return
			}
		}
		check.OnChanged(false)

		sel.OnChanged = func(_ string) {
			if check.Disabled() {
				check.Enable()
			}
		}

		items[i].Widget = container.NewBorder(nil, nil, check, nil, sel)
	}); err != nil {
		a.error(err)
		return
	}

	a.window.SetContent(page("Search results", container.NewVScroll(form)))
	a.renderDialog(nothing{})
}

func (a *application) makeConfirm(manager *playlists.Manager, name string, data *results.Set) *dialog.FormDialog {
	songs, excluded := data.Slice()

	ew := widget.NewMultiLineEntry()
	ew.SetMinRowsVisible(10)
	ew.Text = strings.Join(excluded, "\n")

	items := []*widget.FormItem{
		widget.NewFormItem("Name", widget.NewLabel(name)),
		widget.NewFormItem("Tracks", widget.NewLabel(strconv.Itoa(len(songs)))),
		widget.NewFormItem("Excluded", ew),
	}

	out := dialog.NewForm("Create playlist?", "Yes", "Cancel", items, func(b bool) {
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

	out.Resize(fyne.NewSize(600, 400))

	return out
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

func splitLines(in string) []results.Item {
	var out []results.Item

	dedup := make(map[string]struct{})
	for _, line := range strings.Split(in, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}

		if _, ok := dedup[s]; ok {
			continue
		}
		dedup[s] = struct{}{}

		out = append(out, results.ParseItem(s))
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
