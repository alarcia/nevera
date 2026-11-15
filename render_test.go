package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func testApp(t *testing.T) core.App {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestPages(t *testing.T) {
	app := testApp(t)
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	if err := devSeed(app, now); err != nil {
		t.Fatal(err)
	}
	if err := devSeed(app, now); err == nil {
		t.Fatal("devSeed must refuse a database that already has products")
	}

	f, err := fridgePage(app, now)
	if err != nil {
		t.Fatal(err)
	}
	// Seeded layout: three shelves, then the two drawers sharing one row.
	if len(f.Body) != 4 || len(f.Body[3].Comps) != 2 || len(f.Door) != 3 {
		t.Fatalf("unexpected layout: %d body rows, %d door rows", len(f.Body), len(f.Door))
	}

	var buf bytes.Buffer
	if err := fridgeTpl.ExecuteTemplate(&buf, "base", page{T: ui, Title: "t", Fridge: f}); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{`href="/c/`, `style="--h:`, `class="it k-`, "Balda alta", "PUERTA"} {
		if !strings.Contains(html, want) {
			t.Errorf("fridge page lacks %q", want)
		}
	}

	// A compartment page lists its items with a text urgency label.
	var id string
	for _, r := range f.Body {
		for _, c := range r.Comps {
			if c.Name == "Balda media" {
				id = c.ID
			}
		}
	}
	c, err := compartmentPage(app, id, now)
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := compTpl.ExecuteTemplate(&buf, "base", page{T: ui, Title: "t", Comp: c}); err != nil {
		t.Fatal(err)
	}
	html = buf.String()
	for _, want := range []string{"Leche entera", "<details", "URGENTE", "Caducado", "PRONTO", "Abierto 5 días"} {
		if !strings.Contains(html, want) {
			t.Errorf("compartment page lacks %q", want)
		}
	}

	// Move the milk to another compartment: it is still allowed, and gets a hint.
	other := f.Body[0].Comps[0].ID
	milk, err := app.FindFirstRecordByFilter("items", "product.name = 'Leche entera'")
	if err != nil {
		t.Fatal(err)
	}
	milk.Set("compartment", other)
	if err := app.Save(milk); err != nil {
		t.Fatal(err)
	}
	c, err = compartmentPage(app, other, now)
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := compTpl.ExecuteTemplate(&buf, "base", page{T: ui, Title: "t", Comp: c}); err != nil {
		t.Fatal(err)
	}
	if h := buf.String(); !strings.Contains(h, "Mejor en") || !strings.Contains(h, "Balda media") {
		t.Error("misplaced item must say where it fits better")
	}

	if _, err := compartmentPage(app, "nope", now); err != errNotFound {
		t.Errorf("unknown compartment: got %v, want errNotFound", err)
	}
}
