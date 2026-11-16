package main

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The admin screen is PocketBase's dashboard for the few things the owner
// tunes often: how each type (and each product, which may differ) is drawn and
// sized, its suggested expiry days and shelf, typo fixes and quantities.
// Everything here is plain functions over core.App, so it is tested without HTTP.

var (
	errHasItems = errors.New("product still has stored items")
	errBadValue = errors.New("bad value")
)

// adminOpts are the choices every card shares.
type adminOpts struct {
	T             any // ui, for the templates
	Comps, Sizes  []opt
	Types         []opt // for the type picker of a product
	Shapes, Skins []opt
}

type adminType struct {
	O                 *adminOpts
	ID, Label         string
	Shape, Skin, Size string
	Life              int
	Comp              string // compartment its products are suggested to go in; "" = none
	SizeName          string
	CompName          string
}

type adminItem struct{ ID, Quantity, Where, Expiry string }

// adminProduct keeps the product's own settings apart from what it is drawn
// with: an empty own setting (0 days) means "like its type", see drawnAs.
type adminProduct struct {
	O                     *adminOpts
	ID, Name, Brand       string
	Type                  *adminType
	Shape, Skin, Size     string
	Life                  int
	Comp                  string
	DrawnShape, DrawnSkin string
	Items                 []adminItem
}

type adminView struct {
	O        *adminOpts
	Types    []adminType
	Products []adminProduct
}

func (v *adminView) typeByID(id string) *adminType {
	for i := range v.Types {
		if v.Types[i].ID == id {
			return &v.Types[i]
		}
	}
	return nil
}

func (v *adminView) productByID(id string) *adminProduct {
	for i := range v.Products {
		if v.Products[i].ID == id {
			return &v.Products[i]
		}
	}
	return nil
}

func adminPage(app core.App) (*adminView, error) {
	o := &adminOpts{T: ui}
	var err error
	if o.Sizes, err = options(app, "size", sizeLabels); err != nil {
		return nil, err
	}
	if o.Shapes, err = options(app, "shape", shapeLabels); err != nil {
		return nil, err
	}
	if o.Skins, err = options(app, "skin", skinLabels); err != nil {
		return nil, err
	}

	comps, err := app.FindRecordsByFilter("compartments", "", "row,sort", 0, 0)
	if err != nil {
		return nil, err
	}
	compName := map[string]string{}
	for _, c := range comps {
		compName[c.Id] = c.GetString("name")
		o.Comps = append(o.Comps, opt{c.Id, c.GetString("name")})
	}

	types, err := app.FindRecordsByFilter("types", "", "sort", 0, 0)
	if err != nil {
		return nil, err
	}
	v := &adminView{O: o}
	typeRec := map[string]*core.Record{}
	for _, t := range types {
		typeRec[t.Id] = t
		o.Types = append(o.Types, opt{t.Id, t.GetString("label")})
		at := adminType{O: o, ID: t.Id, Label: t.GetString("label"), Shape: t.GetString("shape"), Skin: t.GetString("skin"),
			Size: orDefault(t.GetString("size"), "m"), Life: t.GetInt("shelfLifeDays")}
		for _, c := range comps {
			if slices.Contains(c.GetStringSlice("types"), t.Id) {
				at.Comp, at.CompName = c.Id, c.GetString("name")
				break
			}
		}
		at.SizeName = sizeLabels[at.Size]
		v.Types = append(v.Types, at)
	}

	items, err := app.FindRecordsByFilter("items", "removedAt = ''", "-created", 0, 0)
	if err != nil {
		return nil, err
	}
	stored := map[string][]adminItem{}
	for _, it := range items {
		ai := adminItem{ID: it.Id, Quantity: it.GetString("quantity"), Where: compName[it.GetString("compartment")]}
		if d := it.GetDateTime("expiresOn"); !d.IsZero() {
			ai.Expiry = dateLabel(calDate(d.Time()))
		}
		stored[it.GetString("product")] = append(stored[it.GetString("product")], ai)
	}

	products, err := app.FindRecordsByFilter("products", "", "-@rowid", 0, 0)
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		tid := p.GetString("type")
		shape, skin, _ := drawnAs(p, typeRec[tid])
		v.Products = append(v.Products, adminProduct{
			O: o, ID: p.Id, Name: p.GetString("name"), Brand: p.GetString("brand"), Type: v.typeByID(tid),
			Shape: p.GetString("shape"), Skin: p.GetString("skin"), Size: p.GetString("size"),
			Life: p.GetInt("shelfLifeDays"), Comp: p.GetString("compartment"),
			DrawnShape: shape, DrawnSkin: skin, Items: stored[p.Id],
		})
	}
	return v, nil
}

// days reads a number of days; empty is 0.
func days(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 3650 {
		return 0, errBadValue
	}
	return n, nil
}

// saveType stores a type's defaults and the compartment its products are
// suggested to go in. Look is a pair: no shape means no skin either.
func saveType(app core.App, id string, f url.Values) error {
	return app.RunInTransaction(func(tx core.App) error {
		r, err := tx.FindRecordById("types", id)
		if err != nil {
			return err
		}
		life, err := days(f.Get("shelfLifeDays"))
		if err != nil {
			return err
		}
		shape, skin := f.Get("shape"), f.Get("skin")
		if shape == "" {
			skin = ""
		}
		r.Set("shape", shape)
		r.Set("skin", skin)
		r.Set("size", f.Get("size"))
		r.Set("shelfLifeDays", life)
		if err := tx.Save(r); err != nil {
			return err
		}
		return recommend(tx, id, f.Get("compartment"))
	})
}

// recommend makes compID (or none, when empty) the compartment recommended for
// a type. It leaves everything alone if that is already the first one, so a
// type recommended in several places (set from the dashboard) keeps them all
// until the owner actually picks another.
func recommend(app core.App, typeID, compID string) error {
	comps, err := app.FindRecordsByFilter("compartments", "", "row,sort", 0, 0)
	if err != nil {
		return err
	}
	cur, known := "", compID == ""
	for _, c := range comps {
		if cur == "" && slices.Contains(c.GetStringSlice("types"), typeID) {
			cur = c.Id
		}
		known = known || c.Id == compID
	}
	if !known {
		return errBadValue
	}
	if compID == cur {
		return nil
	}
	for _, c := range comps {
		ts := c.GetStringSlice("types")
		want, has := c.Id == compID, slices.Contains(ts, typeID)
		if want == has {
			continue
		}
		if want {
			ts = append(ts, typeID)
		} else {
			ts = slices.DeleteFunc(slices.Clone(ts), func(s string) bool { return s == typeID })
		}
		c.Set("types", ts)
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return nil
}

// saveProduct stores a product's name, type and overrides. An empty look, size,
// compartment (or 0 days) hands that setting back to its type.
func saveProduct(app core.App, id string, f url.Values) error {
	r, err := app.FindRecordById("products", id)
	if err != nil {
		return err
	}
	life, err := days(f.Get("shelfLifeDays"))
	if err != nil {
		return err
	}
	shape, skin := f.Get("shape"), f.Get("skin")
	if shape == "" {
		skin = ""
	}
	r.Set("name", strings.TrimSpace(f.Get("name")))
	r.Set("brand", strings.TrimSpace(f.Get("brand")))
	r.Set("type", f.Get("type"))
	r.Set("shape", shape)
	r.Set("skin", skin)
	r.Set("size", f.Get("size"))
	r.Set("shelfLifeDays", life)
	r.Set("compartment", f.Get("compartment"))
	return app.Save(r)
}

// saveItem fixes the quantity of something stored; it returns its product's id.
func saveItem(app core.App, id string, f url.Values) (string, error) {
	it, err := app.FindRecordById("items", id)
	if err != nil {
		return "", err
	}
	it.Set("quantity", strings.TrimSpace(f.Get("quantity")))
	return it.GetString("product"), app.Save(it)
}

// removeItem really deletes something stored: it is for entries made by
// mistake, not for eating or throwing away. It returns its product's id.
func removeItem(app core.App, id string) (string, error) {
	it, err := app.FindRecordById("items", id)
	if err != nil {
		return "", err
	}
	return it.GetString("product"), app.Delete(it)
}

// deleteProduct refuses while anything is stored for the product (or was).
// ponytail: "was" counts items with removedAt set, which the screen does not
// list; decide what happens to a product's history when consume/discard exist.
func deleteProduct(app core.App, id string) error {
	r, err := app.FindRecordById("products", id)
	if err != nil {
		return err
	}
	if n, err := app.CountRecords("items", dbx.HashExp{"product": id}); err != nil {
		return err
	} else if n > 0 {
		return errHasItems
	}
	return app.Delete(r)
}
