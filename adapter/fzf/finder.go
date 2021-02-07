package fzf

import (
	"fmt"
	"os"

	"github.com/mickael-menu/zk/core/note"
	"github.com/mickael-menu/zk/core/style"
	"github.com/mickael-menu/zk/core/zk"
	"github.com/mickael-menu/zk/util/opt"
	stringsutil "github.com/mickael-menu/zk/util/strings"
)

// NoteFinder wraps a note.Finder and filters its result interactively using fzf.
type NoteFinder struct {
	opts   NoteFinderOpts
	finder note.Finder
	styler style.Styler
}

type NoteFinderOpts struct {
	// Indicates whether fzf is opened for every query, even if empty.
	AlwaysFilter bool
	// When non nil, a "create new note from query" binding will be added to
	// fzf to create a note in this directory.
	NewNoteDir *zk.Dir
}

func NewNoteFinder(opts NoteFinderOpts, finder note.Finder, styler style.Styler) *NoteFinder {
	return &NoteFinder{
		opts:   opts,
		finder: finder,
		styler: styler,
	}
}

func (f *NoteFinder) Find(opts note.FinderOpts) ([]note.Match, error) {
	isInteractive, opts := popInteractiveFilter(opts)
	matches, err := f.finder.Find(opts)

	if !isInteractive || err != nil || (!f.opts.AlwaysFilter && len(matches) == 0) {
		return matches, err
	}

	selectedMatches := make([]note.Match, 0)

	zkBin, err := os.Executable()
	if err != nil {
		return selectedMatches, err
	}

	bindings := []Binding{}

	if dir := f.opts.NewNoteDir; dir != nil {
		suffix := ""
		if dir.Name != "" {
			suffix = " in " + dir.Name + "/"
		}

		bindings = append(bindings, Binding{
			Keys:        "Ctrl-N",
			Description: "create a note with the query as title" + suffix,
			Action:      fmt.Sprintf("abort+execute(%s new %s --title {q} < /dev/tty > /dev/tty)", zkBin, dir.Path),
		})
	}

	fzf, err := New(Opts{
		// PreviewCmd: opt.NewString("bat -p --theme Nord --color always {1}"),
		PreviewCmd: opt.NewString(zkBin + " list -f {{raw-content}} {1}"),
		Padding:    2,
		Bindings: bindings,
	})
	if err != nil {
		return selectedMatches, err
	}

	for _, match := range matches {
		fzf.Add([]string{
			match.Path,
			f.styler.MustStyle(match.Title, style.RuleYellow),
			f.styler.MustStyle(stringsutil.JoinLines(match.Body), style.RuleBlack),
		})
	}

	selection, err := fzf.Selection()
	if err != nil {
		return selectedMatches, err
	}

	for _, s := range selection {
		path := s[0]
		for _, m := range matches {
			if m.Path == path {
				selectedMatches = append(selectedMatches, m)
			}
		}
	}

	return selectedMatches, nil
}

func popInteractiveFilter(opts note.FinderOpts) (bool, note.FinderOpts) {
	isInteractive := false
	filters := make([]note.Filter, 0)

	for _, filter := range opts.Filters {
		if f, ok := filter.(note.InteractiveFilter); ok {
			isInteractive = bool(f)
		} else {
			filters = append(filters, filter)
		}
	}

	opts.Filters = filters
	return isInteractive, opts
}
