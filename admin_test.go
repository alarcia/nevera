package main

import (
	"bytes"
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func typeNamed(t *testing.T, app core.App, label string) *core.Record {
	t.Helper()
	r, err := app.FindFirstRecordByFilter("types", "label = {:l}", map[string]any{"l": label})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func compNamed(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()
	r, err := app.FindFirstRecordByFilter("compartments", "name = {:n}", map[string]any{"n": name})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAdminPage(t *testing.T) {
	app := testApp(t)
	if err := devSeed(app, time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	v, err := adminPage(app)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Types) != 14 || len(v.Products) != 22 || len(v.O.Comps) != 8 || len(v.O.Shapes) != 16 {
		t.Fatalf("unexpected counts: %d types, %d products, %d comps", len(v.Types), len(v.Products), len(v.O.Comps))
	}
	// Newest first, each with what is stored; a type knows where it is suggested.
	p := v.Products[0]
	if p.Name != "Refresco de cola" || len(p.Items) != 1 || p.Items[0].Quantity != "2 L" || p.Items[0].Where != "Puerta baja" {
		t.Errorf("first product: %+v", p)
	}
	if p.Type.Label != "Bebida" || p.Type.CompName != "Puerta baja" || p.Type.SizeName != "Mediano" {
		t.Errorf("its type: %+v", p.Type)
	}
}

func TestAdminType(t *testing.T) {
	app := testApp(t)
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	dairy, top := typeNamed(t, app, "Lácteo"), compNamed(t, app, "Balda alta")

	// Defaults and the suggested shelf move together.
	err := saveType(app, dairy.Id, url.Values{"shape": {"bottle"}, "skin": {"oj"}, "size": {"l"}, "shelfLifeDays": {"21"}, "compartment": {top.Id}})
	if err != nil {
		t.Fatal(err)
	}
	dairy, _ = app.FindRecordById("types", dairy.Id)
	if dairy.GetString("shape") != "bottle" || dairy.GetString("skin") != "oj" || dairy.GetString("size") != "l" || dairy.GetInt("shelfLifeDays") != 21 {
		t.Errorf("type not saved: %v", dairy.PublicExport())
	}
	a, err := addPage(app, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Data.Suggest[dairy.Id].Comp; got != top.Id {
		t.Errorf("dairy is suggested in %q, want Balda alta %q", got, top.Id)
	}
	if media := compNamed(t, app, "Balda media"); slices.Contains(media.GetStringSlice("types"), dairy.Id) {
		t.Error("dairy is still recommended in Balda media")
	}

	// Picking the shelf it already has changes nothing, even when a type is
	// recommended in several places.
	low := compNamed(t, app, "Balda baja")
	low.Set("types", append(low.GetStringSlice("types"), dairy.Id))
	if err := app.Save(low); err != nil {
		t.Fatal(err)
	}
	if err := saveType(app, dairy.Id, url.Values{"shape": {"bottle"}, "size": {"l"}, "compartment": {top.Id}}); err != nil {
		t.Fatal(err)
	}
	if low = compNamed(t, app, "Balda baja"); !slices.Contains(low.GetStringSlice("types"), dairy.Id) {
		t.Error("an untouched second recommendation was dropped")
	}
	if dairy, _ = app.FindRecordById("types", dairy.Id); dairy.GetString("skin") != "" || dairy.GetInt("shelfLifeDays") != 0 {
		t.Error("skin and days must follow the form")
	}

	// No shelf at all, then bad input stores nothing.
	if err := saveType(app, dairy.Id, url.Values{"shape": {"bottle"}, "size": {"l"}}); err != nil {
		t.Fatal(err)
	}
	if top = compNamed(t, app, "Balda alta"); slices.Contains(top.GetStringSlice("types"), dairy.Id) {
		t.Error("clearing the shelf left it recommended")
	}
	for name, f := range map[string]url.Values{
		"bad shelf": {"shape": {"jar"}, "size": {"m"}, "compartment": {"nope"}},
		"bad days":  {"shape": {"jar"}, "size": {"m"}, "shelfLifeDays": {"-3"}},
		"words":     {"shape": {"jar"}, "size": {"m"}, "shelfLifeDays": {"soon"}},
		"bad shape": {"shape": {"cube"}, "size": {"m"}},
	} {
		if err := saveType(app, dairy.Id, f); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if dairy, _ = app.FindRecordById("types", dairy.Id); dairy.GetString("shape") != "bottle" {
		t.Error("a failed save changed the type")
	}
}

func TestAdminProduct(t *testing.T) {
	app := testApp(t)
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	dairy, fish := typeNamed(t, app, "Lácteo"), typeNamed(t, app, "Pescado")
	comp := compNamed(t, app, "Balda media")
	if _, err := createItem(app, url.Values{"name": {"Lche"}, "type": {dairy.Id}, "compartment": {comp.Id}, "quantity": {"1 L"}}, now); err != nil {
		t.Fatal(err)
	}
	p, err := app.FindFirstRecordByFilter("products", "name = 'Lche'")
	if err != nil {
		t.Fatal(err)
	}
	it, err := app.FindFirstRecordByFilter("items", "product = {:p}", map[string]any{"p": p.Id})
	if err != nil {
		t.Fatal(err)
	}

	// A typo fix, an own look and an own shelf; the rest still follows the type.
	top := compNamed(t, app, "Balda alta")
	err = saveProduct(app, p.Id, url.Values{"name": {" Leche "}, "brand": {"Marca"}, "type": {dairy.Id}, "shape": {"jar"}, "skin": {"oj"}, "compartment": {top.Id}})
	if err != nil {
		t.Fatal(err)
	}
	v, _ := adminPage(app)
	ap := v.productByID(p.Id)
	if ap.Name != "Leche" || ap.Brand != "Marca" || ap.DrawnShape != "jar" || ap.DrawnSkin != "oj" || ap.Size != "" || ap.Comp != top.Id {
		t.Errorf("product after edit: %+v", ap)
	}
	a, _ := addPage(app, now)
	if i := slices.IndexFunc(a.Data.Products, func(o productOpt) bool { return o.ID == p.Id }); i < 0 || a.Data.Products[i].Comp != top.Id || a.Data.Products[i].Shape != "jar" {
		t.Error("the dialog does not offer the product with its own shelf and look")
	}

	// Handing everything back to the type; a skin without a shape is dropped.
	if err := saveProduct(app, p.Id, url.Values{"name": {"Leche"}, "type": {fish.Id}, "skin": {"oj"}}); err != nil {
		t.Fatal(err)
	}
	p, _ = app.FindRecordById("products", p.Id)
	fish, _ = app.FindRecordById("types", fish.Id)
	if shape, skin, size := drawnAs(p, fish); p.GetString("skin") != "" || shape != fish.GetString("shape") || skin != fish.GetString("skin") || size != "m" {
		t.Errorf("inherited look %s/%s/%s", shape, skin, size)
	}

	for name, f := range map[string]url.Values{
		"no name":  {"type": {dairy.Id}},
		"no type":  {"name": {"x"}},
		"bad type": {"name": {"x"}, "type": {"nope"}},
		"bad size": {"name": {"x"}, "type": {dairy.Id}, "size": {"huge"}},
		"bad days": {"name": {"x"}, "type": {dairy.Id}, "shelfLifeDays": {"-1"}},
		"bad comp": {"name": {"x"}, "type": {dairy.Id}, "compartment": {"nope"}},
	} {
		if err := saveProduct(app, p.Id, f); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if p, _ = app.FindRecordById("products", p.Id); p.GetString("name") != "Leche" {
		t.Error("a failed save changed the product")
	}

	// What is stored: fix the quantity; the product cannot go while it is stored.
	if pid, err := saveItem(app, it.Id, url.Values{"quantity": {" 2 L "}}); err != nil || pid != p.Id {
		t.Fatalf("saveItem: %q %v", pid, err)
	}
	if it, _ = app.FindRecordById("items", it.Id); it.GetString("quantity") != "2 L" {
		t.Errorf("quantity %q", it.GetString("quantity"))
	}
	if err := deleteProduct(app, p.Id); !errors.Is(err, errHasItems) {
		t.Fatalf("deleteProduct with items: %v", err)
	}
	if pid, err := removeItem(app, it.Id); err != nil || pid != p.Id {
		t.Fatalf("removeItem: %q %v", pid, err)
	}
	if err := deleteProduct(app, p.Id); err != nil {
		t.Fatal(err)
	}
	if n, _ := app.CountRecords("products"); n != 0 {
		t.Errorf("%d products left", n)
	}
	if err := deleteProduct(app, p.Id); err == nil {
		t.Error("deleting a missing product must fail")
	}
}

func TestAdminTemplates(t *testing.T) {
	app := testApp(t)
	if err := devSeed(app, time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	v, err := adminPage(app)
	if err != nil {
		t.Fatal(err)
	}
	render := func(name string, data any) string {
		t.Helper()
		var buf bytes.Buffer
		if err := adminTpl.ExecuteTemplate(&buf, name, data); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	html := render("admin", v)
	for _, want := range []string{
		`href="/_/"`, "pocketbase", "Categorías", `id="t-` + v.Types[0].ID + `"`, `id="p-` + v.Products[0].ID + `"`,
		`data-q="Refresco de cola Kora"`, `action="/admin/items/`, ">quitar<", ">Borrar producto<", `id="adm-pick"`,
		`placeholder="90"`, // the product's days field shows its type's value while empty
	} {
		if !strings.Contains(html, want) {
			t.Errorf("admin screen lacks %q", want)
		}
	}
	if n := strings.Count(html, `class="adm-card"`); n != 14+22 {
		t.Errorf("%d cards, want %d", n, 14+22)
	}
	// One product's items block, as the screen swaps it after a quantity edit.
	if items := render("items", v.Products[0]); !strings.Contains(items, `value="2 L"`) || strings.Contains(items, "<article") {
		t.Errorf("items fragment: %s", items)
	}
}

// A product that leaves its look to its type follows the type when the owner
// changes it, all the way to the fridge drawing.
func TestTypeLookReachesTheFridge(t *testing.T) {
	app := testApp(t)
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	dairy, comp := typeNamed(t, app, "Lácteo"), compNamed(t, app, "Balda media")
	if _, err := createItem(app, url.Values{"name": {"Leche"}, "type": {dairy.Id}, "compartment": {comp.Id}}, now); err != nil {
		t.Fatal(err)
	}
	slot := func() slotView {
		t.Helper()
		f, err := fridgePage(app, now)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range f.Body {
			for _, c := range r.Comps {
				if c.Name == "Balda media" && len(c.Slots) == 1 {
					return c.Slots[0]
				}
			}
		}
		t.Fatal("the item is not drawn in Balda media")
		return slotView{}
	}
	if s := slot(); s.Shape != "brick" || s.Skin != "milk" {
		t.Errorf("drawn as %s/%s, want its type's brick/milk", s.Shape, s.Skin)
	}
	if err := saveType(app, dairy.Id, url.Values{"shape": {"bottle"}, "skin": {"oj"}, "size": {"m"}, "compartment": {comp.Id}}); err != nil {
		t.Fatal(err)
	}
	if s := slot(); s.Shape != "bottle" || s.Skin != "oj" {
		t.Errorf("drawn as %s/%s after the type changed, want bottle/oj", s.Shape, s.Skin)
	}
}
