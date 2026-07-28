package app

import (
	"sort"
	"strings"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"go.rockorager.dev/vaxis"
)

const keyHintSep = " - "

type keyHintEntry struct {
	key  string
	desc string
}

// KeyHint displays available keybindings that share a common prefix in
// a floating popover. It implements ui.DrawableInteractive.
type KeyHint struct {
	entries   []keyHintEntry
	prefix    []config.KeyStroke
	maxKeyW   int
	maxDescW  int
	requiredW int
	requiredH int
}

// NewKeyHint creates a KeyHint populated with bindings that match the
// given prefix keystrokes. Duplicate keys are de-duplicated. Entries
// are sorted alphabetically by key.
func NewKeyHint(bindings []*config.Binding, prefix []config.KeyStroke) *KeyHint {
	seen := make(map[string]bool)
	var entries []keyHintEntry
	for _, b := range bindings {
		if len(b.Input) <= len(prefix) {
			continue
		}
		match := true
		for i, s := range prefix {
			if s != b.Input[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		remaining := config.FormatKeyStrokes(b.Input[len(prefix):])
		if seen[remaining] {
			continue
		}
		seen[remaining] = true
		desc := b.Annotation
		if desc == "" {
			desc = formatCommand(b.Output)
		}
		entries = append(entries, keyHintEntry{key: remaining, desc: desc})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})
	maxKeyW := 0
	maxDescW := 0
	for _, e := range entries {
		if len(e.key) > maxKeyW {
			maxKeyW = len(e.key)
		}
		if len(e.desc) > maxDescW {
			maxDescW = len(e.desc)
		}
	}
	if maxDescW > 40 {
		maxDescW = 40
	}
	if maxDescW < 20 {
		maxDescW = 20
	}
	sepW := len(keyHintSep)
	contentW := maxKeyW + sepW + maxDescW
	return &KeyHint{
		entries:   entries,
		prefix:    prefix,
		maxKeyW:   maxKeyW,
		maxDescW:  maxDescW,
		requiredW: contentW + 1,
		requiredH: len(entries),
	}
}

func formatCommand(strokes []config.KeyStroke) string {
	s := config.FormatKeyStrokes(strokes)
	s = strings.TrimPrefix(s, ":")
	s = strings.TrimSuffix(s, "<enter>")
	s = strings.TrimSuffix(s, "<Enter>")
	s = strings.TrimSpace(s)
	return s
}

func (kh *KeyHint) Draw(ctx *ui.Context) {
	if len(kh.entries) == 0 {
		return
	}

	uiConf := SelectedAccountUiConfig()
	defaultStyle := uiConf.GetStyle(config.STYLE_KEYHINT_DEFAULT)
	keyStyle := uiConf.GetStyle(config.STYLE_KEYHINT_KEY)
	descStyle := uiConf.GetStyle(config.STYLE_KEYHINT_DESC)

	w := ctx.Width()
	h := ctx.Height()

	sepW := len(keyHintSep)
	descWidth := w - 1 - kh.maxKeyW - sepW
	if descWidth > kh.maxDescW {
		descWidth = kh.maxDescW
	}
	if descWidth < 1 {
		descWidth = 1
	}

	maxRows := len(kh.entries)
	if maxRows > h {
		maxRows = h
	}

	ctx.Fill(0, 0, w, maxRows, ' ', defaultStyle)

	for i := 0; i < maxRows; i++ {
		e := kh.entries[i]
		ctx.Printf(1, i, keyStyle, "%s", e.key)
		ctx.Printf(1+len(e.key), i, defaultStyle, keyHintSep)
		desc := e.desc
		if len(desc) > descWidth {
			desc = desc[:descWidth]
		}
		ctx.Printf(1+len(e.key)+sepW, i, descStyle, "%s", desc)
	}
}

func (kh *KeyHint) Invalidate() {
	ui.Invalidate()
}

func (kh *KeyHint) Event(event vaxis.Event) bool {
	return false
}

func (kh *KeyHint) Focus(f bool) {}

func (kh *KeyHint) HasEntries() bool {
	return len(kh.entries) > 0
}

func (kh *KeyHint) Prefix() []config.KeyStroke {
	return kh.prefix
}

// RequiredHeight returns the number of content rows needed.
func (kh *KeyHint) RequiredHeight() int {
	return kh.requiredH
}

// RequiredWidth returns the number of content columns needed.
func (kh *KeyHint) RequiredWidth() int {
	return kh.requiredW
}
