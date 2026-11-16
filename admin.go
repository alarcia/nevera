package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
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
	Types         []*adminType // for the category list of a product
	Shapes, Skins []opt
}

// adminType is a category card. The one with no ID is the blank card that
// creates the next category.
type adminType struct {
	O                  *adminOpts
	ID, Label          string
	Shape, Skin, Size  string
	Life               int    // suggested expiry in days: 0 = the same day, -1 = no suggested date
	Comp               string // compartment its products are suggested to go in; "" = none
	SizeName, CompName string
}

// dd is a dropdown: a radio group in a <details>, drawn by the app instead of
// the system's select. An option with From set reads "like the category" and
// shows that category's value (see admin.js).
type dd struct {
	Legend, Kind, Ph, Cur string // Cur is the label of the chosen option; Ph shows when none is
	Opts                  []ddOpt
}

type ddOpt struct {
	Name, Value, Label string
	Checked            bool
	From               string
	T                  *adminType // set on category options: its look and values travel with them
}

// numField is the days stepper.
type numField struct{ ID, Legend, Value, Ph, Less, More string }

func newDD(legend, kind, ph string, opts []ddOpt) dd {
	d := dd{Legend: legend, Kind: kind, Ph: ph, Opts: opts}
	for _, o := range opts {
		if o.Checked {
			d.Cur = o.Label
		}
	}
	return d
}

// choices lists opts as the options of field name with cur checked. A first
// label adds an option for the empty value.
func choices(name, cur string, opts []opt, first, from string) []ddOpt {
	var out []ddOpt
	if first != "" {
		out = append(out, ddOpt{Name: name, Label: first, Checked: cur == "", From: from})
	}
	for _, o := range opts {
		out = append(out, ddOpt{Name: name, Value: o.Key, Label: o.Label, Checked: o.Key == cur})
	}
	return out
}

// like is the label of a "like the category" option, with the category's value.
func like(value string) string {
	if value == "" {
		return ui.AdminLikeType
	}
	return ui.AdminLikeType + " (" + value + ")"
}

func lifeValue(n int) string {
	if n < 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func (t *adminType) New() bool      { return t.ID == "" }
func (t *adminType) DShape() string { return orDefault(t.Shape, "jar") }
func (t *adminType) LifeText() string {
	if t.Life < 0 {
		return ui.AdminNoDate
	}
	return strconv.Itoa(t.Life)
}
func (t *adminType) AsOpt() ddOpt { return ddOpt{Name: "type", Value: t.ID, Label: t.Label, T: t} }
func (t *adminType) SizeDD() dd {
	return newDD(ui.Size, "size", "", choices("size", t.Size, t.O.Sizes, "", ""))
}
func (t *adminType) CompDD() dd {
	return newDD(ui.AdminShelf, "comp", "", choices("compartment", t.Comp, t.O.Comps, ui.AdminNoShelf, ""))
}
func (t *adminType) DaysField() numField {
	return numField{"d-" + orDefault(t.ID, "nt"), ui.AdminDays, lifeValue(t.Life), ui.AdminNoDate, ui.AdminLess, ui.More}
}

type adminItem struct{ ID, Quantity, Where, Expiry string }

// adminProduct keeps the product's own settings apart from what it is drawn
// with: an empty own setting (-1 days) means "like its type", see drawnAs. The
// one with no ID is the blank card that creates the next product; its Type has
// no ID until one is chosen.
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

func (p *adminProduct) New() bool { return p.ID == "" }
func (p *adminProduct) TypeDD() dd {
	opts := make([]ddOpt, len(p.O.Types))
	for i, t := range p.O.Types {
		opts[i] = t.AsOpt()
		opts[i].Checked = t.ID == p.Type.ID && !p.New()
	}
	return newDD(ui.Type, "type", ui.AdminPickType, opts)
}
func (p *adminProduct) SizeDD() dd {
	return newDD(ui.Size, "size", "", choices("size", p.Size, p.O.Sizes, like(p.Type.SizeName), "size"))
}
func (p *adminProduct) CompDD() dd {
	return newDD(ui.AdminShelf, "comp", "", choices("compartment", p.Comp, p.O.Comps, like(p.Type.CompName), "comp"))
}
func (p *adminProduct) DaysField() numField {
	hint := ""
	if p.Type.ID != "" {
		hint = p.Type.LifeText()
	}
	return numField{"d-" + orDefault(p.ID, "np"), ui.AdminDays, lifeValue(p.Life), like(hint), ui.AdminLess, ui.More}
}

type adminView struct {
	O          *adminOpts
	Types      []adminType
	Products   []adminProduct
	NewType    *adminType
	NewProduct *adminProduct
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
	for i := range v.Types {
		o.Types = append(o.Types, &v.Types[i])
	}
	v.NewType = &adminType{O: o, Size: "m", Life: -1}
	v.NewProduct = &adminProduct{O: o, Type: &adminType{O: o, Size: "m", Life: -1}, Life: -1, DrawnShape: "jar"}

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

// days reads a number of days: 0 is the same day, empty is -1 (not set).
func days(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 3650 {
		return 0, errBadValue
	}
	return n, nil
}

// typeFrom fills a category from the form: name, look, size and days. Look is
// a pair: no shape means no skin either.
func typeFrom(r *core.Record, f url.Values) error {
	label := strings.TrimSpace(f.Get("label"))
	life, err := days(f.Get("shelfLifeDays"))
	if err != nil || label == "" {
		return errBadValue
	}
	shape, skin := f.Get("shape"), f.Get("skin")
	if shape == "" {
		skin = ""
	}
	r.Set("label", label)
	r.Set("shape", shape)
	r.Set("skin", skin)
	r.Set("size", f.Get("size"))
	r.Set("shelfLifeDays", life)
	return nil
}

// saveType stores a category's name and defaults, and the compartment its
// products are suggested to go in.
func saveType(app core.App, id string, f url.Values) error {
	return app.RunInTransaction(func(tx core.App) error {
		r, err := tx.FindRecordById("types", id)
		if err != nil {
			return err
		}
		if err := typeFrom(r, f); err != nil {
			return err
		}
		if err := tx.Save(r); err != nil {
			return err
		}
		return recommend(tx, id, f.Get("compartment"))
	})
}

var notKey = regexp.MustCompile(`[^a-z0-9]+`)

// freeKey makes the stable key of a new category from its name (accents
// folded, "_" between words), numbered when another one has it.
func freeKey(app core.App, label string) (string, error) {
	fold := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "ç", "c")
	base := strings.Trim(notKey.ReplaceAllString(fold.Replace(strings.ToLower(label)), "_"), "_")
	if base == "" {
		base = "type"
	}
	for i := 1; ; i++ {
		key := base
		if i > 1 {
			key = fmt.Sprintf("%s_%d", base, i)
		}
		_, err := app.FindFirstRecordByFilter("types", "key = {:k}", dbx.Params{"k": key})
		if errors.Is(err, sql.ErrNoRows) {
			return key, nil
		} else if err != nil {
			return "", err
		}
	}
}

// createType adds a category after the existing ones and returns its id.
func createType(app core.App, f url.Values) (id string, err error) {
	err = app.RunInTransaction(func(tx core.App) error {
		c, err := tx.FindCollectionByNameOrId("types")
		if err != nil {
			return err
		}
		r := core.NewRecord(c)
		if err := typeFrom(r, f); err != nil {
			return err
		}
		key, err := freeKey(tx, r.GetString("label"))
		if err != nil {
			return err
		}
		sort := 1
		if last, err := tx.FindRecordsByFilter(c, "", "-sort", 1, 0); err != nil {
			return err
		} else if len(last) > 0 {
			sort = last[0].GetInt("sort") + 1
		}
		r.Set("key", key)
		r.Set("sort", sort)
		if err := tx.Save(r); err != nil {
			return err
		}
		id = r.Id
		return recommend(tx, id, f.Get("compartment"))
	})
	return id, err
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

// productFrom fills a product from the form: name, type and overrides. An
// empty look, size, compartment (or empty days, -1) hands that setting back to
// its type. The record's own validation rejects a missing name or type.
func productFrom(r *core.Record, f url.Values) error {
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
	return nil
}

func saveProduct(app core.App, id string, f url.Values) error {
	r, err := app.FindRecordById("products", id)
	if err != nil {
		return err
	}
	if err := productFrom(r, f); err != nil {
		return err
	}
	return app.Save(r)
}

// createProduct adds a product to the catalog and returns its id.
func createProduct(app core.App, f url.Values) (string, error) {
	c, err := app.FindCollectionByNameOrId("products")
	if err != nil {
		return "", err
	}
	r := core.NewRecord(c)
	if err := productFrom(r, f); err != nil {
		return "", err
	}
	return r.Id, app.Save(r)
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
