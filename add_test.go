package main

import (
	"bytes"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSuggest(t *testing.T) {
	comps := []compFit{
		{ID: "top", Name: "Balda alta", Types: []string{"left"}, Fill: 30},
		{ID: "mid", Name: "Balda media", Types: []string{"dairy"}, Fill: 120},
		{ID: "drw", Name: "Cajón", Types: []string{"veg"}, Fill: 10},
		{ID: "egg", Name: "Puerta alta", Types: []string{"eggs"}, Fill: 0},
	}
	cases := []struct{ typ, want, why string }{
		{"left", "top", "Es el sitio pensado para x."},
		{"dairy", "egg", "Balda media está llena; Puerta alta es la que tiene más hueco."},
		{"veg", "drw", "Es el sitio pensado para x."},
		{"none", "egg", "Nada está pensado para x; Puerta alta es la que tiene más hueco."},
	}
	for _, c := range cases {
		got := suggest(comps, c.typ, "X")
		if got.Comp != c.want || got.Why != c.why {
			t.Errorf("suggest(%s) = %+v, want %s %q", c.typ, got, c.want, c.why)
		}
	}
	// The first recommended compartment that is not full wins.
	full := []compFit{{ID: "a", Name: "A", Types: []string{"veg"}, Fill: 100}, {ID: "b", Name: "B", Types: []string{"veg"}, Fill: 50}}
	if got := suggest(full, "veg", "X"); got.Comp != "b" || got.Why != "Es el sitio pensado para x." {
		t.Errorf("skip the full one: got %+v", got)
	}
	if got := suggest(nil, "veg", "X"); got.Comp != "" {
		t.Errorf("no compartments: got %+v", got)
	}
}

func TestCreateItem(t *testing.T) {
	app := testApp(t)
	now := time.Date(2026, 11, 14, 23, 30, 0, 0, time.UTC) // already the 15th in Andorra
	a, err := addPage(app, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.Data.Today != "2026-11-15" || len(a.Types) == 0 || len(a.Shapes) != 16 || len(a.Sizes) != 5 {
		t.Fatalf("unexpected add view: today %s, %d types, %d shapes", a.Data.Today, len(a.Types), len(a.Shapes))
	}
	dairy := a.Types[4]
	if dairy.Label != "Lácteo" || dairy.Shape != "brick" {
		t.Fatalf("unexpected type order: %+v", dairy)
	}
	comp := a.Data.Suggest[dairy.ID].Comp
	if comp == "" {
		t.Fatal("no suggestion for dairy")
	}

	// A new product.
	got, err := createItem(app, url.Values{
		"name": {" Leche "}, "brand": {"Marca"}, "type": {dairy.ID}, "size": {"l"},
		"shape": {"brick"}, "skin": {"milk"}, "compartment": {comp},
		"expiresOn": {"2026-11-22"}, "opened": {"on"}, "quantity": {"1 L"},
	}, now)
	if err != nil || got != comp {
		t.Fatalf("createItem: %v (comp %q)", err, got)
	}
	it, err := app.FindFirstRecordByFilter("items", "product.name = 'Leche'")
	if err != nil {
		t.Fatal(err)
	}
	if d := it.GetDateTime("expiresOn").Time(); d.Format(time.DateOnly) != "2026-11-22" {
		t.Errorf("expiresOn = %v", d)
	}
	if d := it.GetDateTime("openedAt").Time(); d.Format(time.DateOnly) != "2026-11-15" {
		t.Errorf("openedAt = %v, want Andorra's today", d)
	}

	// Re-adding it from the catalog creates no second product.
	if _, err := createItem(app, url.Values{"product": {it.GetString("product")}, "compartment": {comp}}, now); err != nil {
		t.Fatal(err)
	}
	if n, _ := app.CountRecords("products"); n != 1 {
		t.Errorf("%d products, want 1", n)
	}
	if n, _ := app.CountRecords("items"); n != 2 {
		t.Errorf("%d items, want 2", n)
	}

	// Bad input stores nothing.
	for name, f := range map[string]url.Values{
		"no name":      {"type": {dairy.ID}, "size": {"m"}, "compartment": {comp}},
		"bad size":     {"name": {"x"}, "type": {dairy.ID}, "size": {"huge"}, "compartment": {comp}},
		"bad date":     {"name": {"x"}, "type": {dairy.ID}, "size": {"m"}, "compartment": {comp}, "expiresOn": {"mañana"}},
		"no such comp": {"name": {"x"}, "type": {dairy.ID}, "size": {"m"}, "compartment": {"nope"}},
		"no such prod": {"product": {"nope"}, "compartment": {comp}},
	} {
		if _, err := createItem(app, f, now); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if n, _ := app.CountRecords("products"); n != 1 {
		t.Errorf("failed saves left %d products, want 1", n)
	}

	// The dialog renders with its data on every page.
	a, err = addPage(app, now)
	if err != nil {
		t.Fatal(err)
	}
	f, err := fridgePage(app, now)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := fridgeTpl.ExecuteTemplate(&buf, "base", page{T: ui, Title: "t", Fridge: f, Add: a}); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{`command="show-modal"`, `<dialog id="add"`, `name="compartment" value="` + comp, `"Today":"2026-11-15"`, `"Name":"Leche"`, `class="it k-melon`, `name="again"`, "Más campos"} {
		if !strings.Contains(html, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(html, `class="qk`) {
		t.Error("the expiry shortcuts are back")
	}

	// Only the name, type and a place are needed: look and size are left to the type.
	if _, err := createItem(app, url.Values{"name": {"Pan"}, "type": {dairy.ID}, "compartment": {comp}}, now); err != nil {
		t.Fatal(err)
	}
	pan, err := app.FindFirstRecordByFilter("products", "name = 'Pan'")
	if err != nil {
		t.Fatal(err)
	}
	typ, err := app.FindRecordById("types", dairy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if shape, skin, size := drawnAs(pan, typ); pan.GetString("shape") != "" || shape != "brick" || skin != "milk" || size != "m" {
		t.Errorf("quick add drawn as %s/%s/%s, want its type's brick/milk/m", shape, skin, size)
	}
	// A product's own look wins, and its skin goes with its shape.
	pan.Set("shape", "jar")
	if shape, skin, _ := drawnAs(pan, typ); shape != "jar" || skin != "" {
		t.Errorf("own look drawn as %s/%s, want jar with no skin", shape, skin)
	}
}
