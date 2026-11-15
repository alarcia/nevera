package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Layout of the cutaway, configured from the dashboard:
// compartments sharing a row sit side by side (two drawers), ordered by sort;
// a full-width drawer between shelves is just a drawer alone in its row.
// Door compartments are drawn in the door column, ordered by row.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("compartments")
		if err != nil {
			return err
		}
		c.Fields.Add(
			&core.NumberField{Name: "row", OnlyInt: true, Min: ptr(1), Help: "1 = top. Same row = side by side"},
			&core.NumberField{Name: "height", OnlyInt: true, Help: "Usable height in cm; the cutaway is drawn to scale. 0 = default"},
		)
		c.Fields.GetByName("sort").(*core.NumberField).Help = "Left to right within the row"
		if err := app.Save(c); err != nil {
			return err
		}

		// Give the seeded mockup layout a sensible arrangement.
		recs, err := app.FindAllRecords(c)
		if err != nil {
			return err
		}
		for _, r := range recs {
			switch r.GetString("kind") {
			case "shelf":
				r.Set("row", r.GetInt("sort"))
				r.Set("height", 20)
			case "drawer":
				r.Set("row", 4)
				r.Set("height", 18)
			case "door":
				r.Set("row", r.GetInt("sort")-5)
				r.Set("height", 12)
			}
			if err := app.Save(r); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("compartments")
		if err != nil {
			return err
		}
		c.Fields.RemoveByName("row")
		c.Fields.RemoveByName("height")
		return app.Save(c)
	})
}
