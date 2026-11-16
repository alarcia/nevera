package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Shelf life used to read "0 = not set". But some things really do expire the
// day they are bought, so 0 is now a value (the same day) and "not set" is -1:
// no suggested date for a type, "use the type's" for a product.
//
// PocketBase numbers are never null, so a record made without this field (the
// dashboard's "new record") gets 0, i.e. same day. The Help text says so.
func init() {
	help := map[string][2]string{ // collection -> up, down
		"types":    {"Suggested expiry = today + this many days; 0 = the same day. -1 = no suggested date", "Suggested expiry = today + this. 0 = no expiry date"},
		"products": {"Overrides the type's; 0 = the same day. -1 = use the type's", "Overrides the type's. 0 = use the type's"},
	}
	// rewrite changes the field's limits and help, then moves every `from` to `to`.
	rewrite := func(app core.App, from, to, h int, min, max *float64) error {
		for _, name := range []string{"types", "products"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			f := c.Fields.GetByName("shelfLifeDays").(*core.NumberField)
			f.Min, f.Max, f.Help = min, max, help[name][h]
			if err := app.Save(c); err != nil {
				return err
			}
			recs, err := app.FindRecordsByFilter(c, "shelfLifeDays = {:v}", "", 0, 0, dbx.Params{"v": from})
			if err != nil {
				return err
			}
			for _, r := range recs {
				r.Set("shelfLifeDays", to)
				if err := app.Save(r); err != nil {
					return err
				}
			}
		}
		return nil
	}
	m.Register(func(app core.App) error {
		return rewrite(app, 0, -1, 0, ptr(-1), ptr(3650))
	}, func(app core.App) error {
		return rewrite(app, -1, 0, 1, nil, nil) // a same-day 0 is lost: it reads "not set" again
	})
}
