package main

import (
	"errors"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// cutawayMax bounds how many 3D items are drawn per compartment in the
// cutaway (each one is ~10 DOM nodes); the rest are implied by the "+N".
const cutawayMax = 8

const defaultHeight = 20 // cm, when a compartment has none set

var errNotFound = errors.New("not found")

type slotView struct {
	Shape, Skin string
	Back        bool // every third item is pushed back for depth
}

type itemView struct {
	Name, Brand, Type, Quantity string
	Shape, Skin                 string
	LevelClass                  string // "u0".."u2" or "ok"
	LevelText                   string // text label: urgency never relies on colour alone
	Chip, ExpLong, OpenLabel    string
	HasPct                      bool
	Pct                         int
	Misplaced                   bool
	Best                        string
}

type compView struct {
	ID, Name, Kind, Why string
	Recommended         string
	Row, Height         int
	Count               int
	CountLabel          string
	HasFill             bool
	Fill                int // percent, may exceed 100
	FillBar             int // clamped for drawing
	Slots               []slotView
	More                int
	Items               []itemView
}

type rowView struct {
	Height int
	Gap    bool // empty space (cm) that keeps door and body to the same scale
	Comps  []compView
}

type fridgeView struct {
	Unit       string
	Body, Door []rowView
}

type stock struct {
	unit    *core.Record
	comps   []*core.Record
	byComp  map[string][]*core.Record
	now     time.Time
	typeFor map[string]string // type id -> compartment id (first recommended)
}

// loadStock reads the first unit with its compartments and everything stored.
func loadStock(app core.App, now time.Time) (*stock, error) {
	units, err := app.FindRecordsByFilter("units", "", "sort", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, errNotFound
	}
	s := &stock{unit: units[0], byComp: map[string][]*core.Record{}, now: now, typeFor: map[string]string{}}

	s.comps, err = app.FindRecordsByFilter("compartments", "unit = {:u}", "row,sort", 0, 0, dbx.Params{"u": s.unit.Id})
	if err != nil {
		return nil, err
	}
	if errs := app.ExpandRecords(s.comps, []string{"types"}, nil); len(errs) > 0 {
		return nil, firstErr(errs)
	}
	for _, c := range s.comps {
		for _, t := range c.GetStringSlice("types") {
			if _, ok := s.typeFor[t]; !ok {
				s.typeFor[t] = c.Id
			}
		}
	}

	items, err := app.FindRecordsByFilter("items", "compartment.unit = {:u} && removedAt = ''", "-created", 0, 0, dbx.Params{"u": s.unit.Id})
	if err != nil {
		return nil, err
	}
	if errs := app.ExpandRecords(items, []string{"product.type"}, nil); len(errs) > 0 {
		return nil, firstErr(errs)
	}
	for _, it := range items {
		c := it.GetString("compartment")
		s.byComp[c] = append(s.byComp[c], it)
	}
	return s, nil
}

func firstErr(errs map[string]error) error {
	for _, e := range errs {
		return e
	}
	return nil
}

func (s *stock) compName(id string) string {
	for _, c := range s.comps {
		if c.Id == id {
			return c.GetString("name")
		}
	}
	return ""
}

func (s *stock) itemView(it *core.Record, compID string) itemView {
	p := it.ExpandedOne("product")
	typ := p.ExpandedOne("type")
	maxOpen := p.GetInt("maxDaysOpen")
	if maxOpen == 0 {
		maxOpen = typ.GetInt("maxDaysOpen")
	}
	st := itemStatus(s.now, it.GetDateTime("expiresOn").Time(), it.GetDateTime("openedAt").Time(), maxOpen)

	v := itemView{
		Name: p.GetString("name"), Brand: p.GetString("brand"), Type: typ.GetString("label"),
		Quantity: it.GetString("quantity"),
		Shape:    orDefault(p.GetString("shape"), "jar"), Skin: p.GetString("skin"),
		Chip: st.Chip, ExpLong: st.ExpLong, OpenLabel: st.OpenLabel,
		LevelClass: "ok", LevelText: "",
		HasPct: it.GetInt("remainingPct") > 0, Pct: it.GetInt("remainingPct"),
	}
	switch st.Level {
	case Urgent:
		v.LevelClass, v.LevelText = "u0", ui.Urgent
	case Soon:
		v.LevelClass, v.LevelText = "u1", ui.Soon
	case Watch:
		v.LevelClass, v.LevelText = "u2", ui.Watch
	}
	// A hint, never a constraint: only when the type has a recommended home elsewhere.
	if best, ok := s.typeFor[typ.Id]; ok && best != compID {
		v.Misplaced, v.Best = true, s.compName(best)
	}
	return v
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (s *stock) compView(c *core.Record, withItems bool) compView {
	items := s.byComp[c.Id]
	points := 0
	for _, it := range items {
		points += sizeWeight[it.ExpandedOne("product").GetString("size")]
	}
	var rec []string
	for _, t := range c.ExpandedAll("types") {
		rec = append(rec, t.GetString("label"))
	}
	v := compView{
		ID: c.Id, Name: c.GetString("name"), Kind: c.GetString("kind"), Why: c.GetString("why"),
		Recommended: strings.Join(rec, ", "),
		Row:         c.GetInt("row"), Height: orHeight(c.GetInt("height")),
		Count: len(items), CountLabel: countLabel(len(items)),
	}
	if capacity := c.GetInt("capacity"); capacity > 0 {
		v.HasFill, v.Fill = true, fillPercent(points, capacity)
		v.FillBar = min(v.Fill, 100)
	}
	for i, it := range items {
		if withItems {
			v.Items = append(v.Items, s.itemView(it, c.Id))
			continue
		}
		if i == cutawayMax {
			v.More = len(items) - cutawayMax
			break
		}
		p := it.ExpandedOne("product")
		v.Slots = append(v.Slots, slotView{Shape: orDefault(p.GetString("shape"), "jar"), Skin: p.GetString("skin"), Back: i%3 == 1})
	}
	return v
}

func orHeight(h int) int {
	if h <= 0 {
		return defaultHeight
	}
	return h
}

// groupRows joins consecutive compartments sharing a row number (already
// sorted by row, sort) into one row; a compartment alone in its row is full width.
func groupRows(comps []compView) []rowView {
	var rows []rowView
	for i, c := range comps {
		if i == 0 || c.Row != comps[i-1].Row {
			rows = append(rows, rowView{})
		}
		r := &rows[len(rows)-1]
		r.Comps = append(r.Comps, c)
		r.Height = max(r.Height, c.Height)
	}
	return rows
}

func totalHeight(rows []rowView) (t int) {
	for _, r := range rows {
		t += r.Height
	}
	return
}

// withGap puts the spare height of a shorter column before its last row, so
// small door shelves hang from the top and the bottle shelf stands at the
// bottom, both drawn to the same scale as the body.
func withGap(rows []rowView, bodyTotal int) []rowView {
	gap := bodyTotal - totalHeight(rows)
	if len(rows) < 2 || gap <= 0 {
		return rows
	}
	last := len(rows) - 1
	return append(rows[:last:last], rowView{Gap: true, Height: gap}, rows[last])
}

func fridgePage(app core.App, now time.Time) (*fridgeView, error) {
	s, err := loadStock(app, now)
	if err != nil {
		return nil, err
	}
	var body, door []compView
	for _, c := range s.comps {
		if c.GetString("kind") == "door" {
			door = append(door, s.compView(c, false))
		} else {
			body = append(body, s.compView(c, false))
		}
	}
	bodyRows, doorRows := groupRows(body), groupRows(door)
	return &fridgeView{Unit: s.unit.GetString("name"), Body: bodyRows, Door: withGap(doorRows, totalHeight(bodyRows))}, nil
}

func compartmentPage(app core.App, id string, now time.Time) (*compView, error) {
	s, err := loadStock(app, now)
	if err != nil {
		return nil, err
	}
	for _, c := range s.comps {
		if c.Id == id {
			v := s.compView(c, true)
			return &v, nil
		}
	}
	return nil, errNotFound
}
