package main

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type opt struct{ Key, Label string }

type typeOpt struct {
	ID, Label   string
	Life        int // suggested expiry in days: 0 = the same day, -1 = none
	StartsOpen  bool
	Shape, Skin string
	Size        string
}

// productOpt is a catalog product as the dialog re-adds it: Shape and Skin are
// what it is drawn with (its own or its type's), Comp its own suggested
// compartment ("" = its type's), Life its own days (-1 = its type's).
type productOpt struct {
	ID, Name, Brand, Type, Shape, Skin, Comp string
	Life                                     int
}

type suggestion struct{ Comp, Why string }

// addView feeds the add dialog, present on every page.
type addView struct {
	Body, Door                []rowView
	Types                     []typeOpt
	Sizes, Shapes, Skins      []opt
	Data                      addData // read by add.js
	DefaultShape, DefaultSkin string
	NewProduct, SavedIn       string
}

type addData struct {
	Today    string // yyyy-mm-dd in Andorra
	Types    []typeOpt
	Products []productOpt
	Suggest  map[string]suggestion // type id -> where it goes
}

// compFit is what the put-away heuristic needs to know about a compartment.
type compFit struct {
	ID, Name string
	Types    []string
	Fill     int // percent; 0 when the compartment has no capacity set
}

// suggest picks a compartment for an item of a type, with the reason shown
// to the user. It is a hint: any compartment is allowed. The size of the
// product plays no part.
//
//  1. The first compartment recommended for the type that is not full.
//  2. Otherwise the least full compartment.
func suggest(comps []compFit, typeID, typeLabel string) suggestion {
	var home, emptiest *compFit
	for i := range comps {
		c := &comps[i]
		if slices.Contains(c.Types, typeID) {
			if home == nil {
				home = c
			}
			if c.Fill < 100 {
				return suggestion{c.ID, reason(ui.SugHome, c, typeLabel, home)}
			}
		}
		if emptiest == nil || c.Fill < emptiest.Fill {
			emptiest = c
		}
	}
	switch {
	case emptiest == nil:
		return suggestion{}
	case home == nil:
		return suggestion{emptiest.ID, reason(ui.SugNoHome, emptiest, typeLabel, nil)}
	}
	return suggestion{emptiest.ID, reason(ui.SugFull, emptiest, typeLabel, home)}
}

func reason(format string, c *compFit, typeLabel string, home *compFit) string {
	homeName := ""
	if home != nil {
		homeName = home.Name
	}
	return fmt.Sprintf(format, c.Name, strings.ToLower(typeLabel), homeName)
}

func options(app core.App, field string, labels map[string]string) ([]opt, error) {
	c, err := app.FindCollectionByNameOrId("products")
	if err != nil {
		return nil, err
	}
	f, ok := c.Fields.GetByName(field).(*core.SelectField)
	if !ok {
		return nil, fmt.Errorf("products.%s is not a select field", field)
	}
	out := make([]opt, len(f.Values))
	for i, v := range f.Values {
		out[i] = opt{v, orDefault(labels[v], v)}
	}
	return out, nil
}

func addPage(app core.App, now time.Time) (*addView, error) {
	s, err := loadStock(app, now)
	if err != nil {
		return nil, err
	}
	v := &addView{
		DefaultShape: "jar", NewProduct: ui.NewProduct, SavedIn: ui.SavedIn,
		Data: addData{Today: today(now).Format(time.DateOnly), Suggest: map[string]suggestion{}},
	}
	v.Body, v.Door = s.layout()
	if v.Sizes, err = options(app, "size", sizeLabels); err != nil {
		return nil, err
	}
	if v.Shapes, err = options(app, "shape", shapeLabels); err != nil {
		return nil, err
	}
	if v.Skins, err = options(app, "skin", skinLabels); err != nil {
		return nil, err
	}

	types, err := app.FindRecordsByFilter("types", "", "sort", 0, 0)
	if err != nil {
		return nil, err
	}
	var fits []compFit
	for _, c := range s.comps {
		cv := s.compView(c, false)
		fits = append(fits, compFit{c.Id, cv.Name, c.GetStringSlice("types"), cv.Fill})
	}
	typeByID := map[string]*core.Record{}
	for _, t := range types {
		typeByID[t.Id] = t
		to := typeOpt{t.Id, t.GetString("label"), t.GetInt("shelfLifeDays"), t.GetBool("startsOpen"),
			orDefault(t.GetString("shape"), "jar"), t.GetString("skin"), orDefault(t.GetString("size"), "m")}
		v.Types = append(v.Types, to)
		v.Data.Suggest[t.Id] = suggest(fits, t.Id, to.Label)
	}
	v.Data.Types = v.Types

	products, err := app.FindRecordsByFilter("products", "", "name", 0, 0)
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		shape, skin, _ := drawnAs(p, typeByID[p.GetString("type")])
		v.Data.Products = append(v.Data.Products, productOpt{
			p.Id, p.GetString("name"), p.GetString("brand"), p.GetString("type"), shape, skin, p.GetString("compartment"), p.GetInt("shelfLifeDays"),
		})
	}
	return v, nil
}

var errBadDate = errors.New("bad expiry date")

// createItem stores what the add form posted: an existing catalog product
// (field "product") or a new one, placed in a compartment. A new product's
// shape, skin and size are only posted when the user chose them; left empty it
// is drawn like its type (see drawnAs). Record validation
// (required fields, select values, related ids) is PocketBase's.
// It returns the compartment id.
func createItem(app core.App, f url.Values, now time.Time) (string, error) {
	field := func(k string) string { return strings.TrimSpace(f.Get(k)) }
	var exp time.Time
	if d := field("expiresOn"); d != "" {
		var err error
		if exp, err = time.Parse(time.DateOnly, d); err != nil {
			return "", errBadDate
		}
	}
	return field("compartment"), app.RunInTransaction(func(tx core.App) error {
		pid := field("product")
		if pid == "" {
			pc, err := tx.FindCollectionByNameOrId("products")
			if err != nil {
				return err
			}
			p := core.NewRecord(pc)
			p.Load(map[string]any{
				"name": field("name"), "brand": field("brand"), "type": field("type"),
				"size": field("size"), "shape": field("shape"), "skin": field("skin"),
				"shelfLifeDays": -1, // like its type
			})
			if err := tx.Save(p); err != nil {
				return err
			}
			pid = p.Id
		}
		ic, err := tx.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		it := core.NewRecord(ic)
		it.Load(map[string]any{"product": pid, "compartment": field("compartment"), "quantity": field("quantity")})
		if !exp.IsZero() {
			it.Set("expiresOn", exp)
		}
		if f.Get("opened") != "" {
			it.Set("openedAt", today(now))
		}
		return tx.Save(it)
	})
}
