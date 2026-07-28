package app

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/config"
	"go.rockorager.dev/vaxis"
)

func stroke(key rune) config.KeyStroke {
	return config.KeyStroke{Key: key}
}

func binding(input, output []config.KeyStroke, annotation string) *config.Binding {
	return &config.Binding{Input: input, Output: output, Annotation: annotation}
}

func TestNewKeyHintEmptyPrefix(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold"),
		binding([]config.KeyStroke{stroke('z'), stroke('o')}, nil, "unfold"),
		binding([]config.KeyStroke{stroke('g'), stroke('g')}, nil, "top"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('z')})
	if len(kh.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(kh.entries))
	}
	if kh.entries[0].key != "c" {
		t.Errorf("first key: got %q, want %q", kh.entries[0].key, "c")
	}
	if kh.entries[0].desc != "fold" {
		t.Errorf("first desc: got %q, want %q", kh.entries[0].desc, "fold")
	}
	if kh.entries[1].key != "o" {
		t.Errorf("second key: got %q, want %q", kh.entries[1].key, "o")
	}
}

func TestNewKeyHintDeduplication(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold"),
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold duplicate"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('z')})
	if len(kh.entries) != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d", len(kh.entries))
	}
	if kh.entries[0].desc != "fold" {
		t.Errorf("desc: got %q, want %q", kh.entries[0].desc, "fold")
	}
}

func TestNewKeyHintAnnotationFallback(t *testing.T) {
	output := []config.KeyStroke{
		{Key: ':'},
		{Key: 'n'},
		{Key: 'e'},
		{Key: 'x'},
		{Key: 't'},
		{Key: vaxis.KeyEnter},
	}
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('g'), stroke('n')}, output, ""),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('g')})
	if len(kh.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(kh.entries))
	}
	if kh.entries[0].desc != "next" {
		t.Errorf("desc fallback: got %q, want %q", kh.entries[0].desc, "next")
	}
}

func TestNewKeyHintSorting(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('z'), stroke('b')}, nil, "bottom"),
		binding([]config.KeyStroke{stroke('z'), stroke('a')}, nil, "toggle"),
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('z')})
	if len(kh.entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(kh.entries))
	}
	want := []string{"a", "b", "c"}
	for i, e := range kh.entries {
		if e.key != want[i] {
			t.Errorf("entry %d: got key %q, want %q", i, e.key, want[i])
		}
	}
}

func TestNewKeyHintNoMatch(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('g'), stroke('g')}, nil, "top"),
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('r')})
	if len(kh.entries) != 0 {
		t.Fatalf("expected 0 entries for non-matching prefix, got %d", len(kh.entries))
	}
}

func TestNewKeyHintMultiKeyPrefix(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('r'), stroke('t'), stroke('q')}, nil, "quick ack"),
		binding([]config.KeyStroke{stroke('r'), stroke('t'), stroke('f')}, nil, "follow up"),
		binding([]config.KeyStroke{stroke('r'), stroke('r')}, nil, "reply all"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('r'), stroke('t')})
	if len(kh.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(kh.entries))
	}
	if kh.entries[0].key != "f" {
		t.Errorf("first key: got %q, want %q", kh.entries[0].key, "f")
	}
	if kh.entries[1].key != "q" {
		t.Errorf("second key: got %q, want %q", kh.entries[1].key, "q")
	}
}

func TestNewKeyHintExcludesExactMatch(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('g')}, nil, "top"),
		binding([]config.KeyStroke{stroke('g'), stroke('g')}, nil, "gg"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('g')})
	if len(kh.entries) != 1 {
		t.Fatalf("expected 1 entry (exact match excluded), got %d", len(kh.entries))
	}
	if kh.entries[0].key != "g" {
		t.Errorf("key: got %q, want %q", kh.entries[0].key, "g")
	}
}

func TestNewKeyHintSizing(t *testing.T) {
	bindings := []*config.Binding{
		binding([]config.KeyStroke{stroke('z'), stroke('c')}, nil, "fold"),
		binding([]config.KeyStroke{stroke('z'), stroke('o')}, nil, "unfold"),
	}
	kh := NewKeyHint(bindings, []config.KeyStroke{stroke('z')})
	if kh.RequiredHeight() != 2 {
		t.Errorf("height: got %d, want 2 (2 content rows)", kh.RequiredHeight())
	}
	wantW := 1 + 3 + 20 + 1
	if kh.RequiredWidth() != wantW {
		t.Errorf("width: got %d, want %d", kh.RequiredWidth(), wantW)
	}
}

func TestNewKeyHintEmptyBindings(t *testing.T) {
	kh := NewKeyHint(nil, []config.KeyStroke{stroke('z')})
	if len(kh.entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(kh.entries))
	}
	if kh.RequiredHeight() != 0 {
		t.Errorf("height: got %d, want 0 (no entries)", kh.RequiredHeight())
	}
}
