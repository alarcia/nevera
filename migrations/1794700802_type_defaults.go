package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Per-type defaults for how things are drawn and sized, and the overrides a
// product can set on top. A product's own shape/skin, size, days and
// compartment win when set; empty (0 for days) means "use the type's".
func init() {
	// type key -> shape, skin: what products of that type look like by default.
	looks := map[string][2]string{
		"leftovers": {"tupper", "tup1"}, "ready_meal": {"tray", "lasagna"},
		"vegetable": {"leaf", "lettuce"}, "fruit": {"round", "apple"},
		"dairy": {"brick", "milk"}, "cheese": {"block", "cheese"},
		"cold_cuts": {"pack", "ham"}, "sauce": {"squeeze", "ketchup"},
		"preserve": {"jar", "pickle"}, "meat": {"tray", "beef"},
		"fish": {"tray", "hake"}, "eggs": {"carton", "egg"},
		"butter": {"block", "butter"}, "drink": {"bottlel", "water"},
	}
	m.Register(func(app core.App) error {
		types, err := app.FindCollectionByNameOrId("types")
		if err != nil {
			return err
		}
		types.Fields.Add(
			&core.SelectField{Name: "shape", MaxSelect: 1, Values: shapes, Help: "3D shape its products are drawn with, unless a product sets its own"},
			&core.SelectField{Name: "skin", MaxSelect: 1, Values: skins, Help: "Colours its products are drawn with, unless a product sets its own"},
			&core.SelectField{Name: "size", MaxSelect: 1, Values: sizes, Help: "Size of its products, unless a product sets its own"},
		)
		if err := app.Save(types); err != nil {
			return err
		}

		comps, err := app.FindCollectionByNameOrId("compartments")
		if err != nil {
			return err
		}
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		size := products.Fields.GetByName("size").(*core.SelectField)
		size.Required, size.Help = false, "Own size. Empty = the type's"
		products.Fields.GetByName("shape").(*core.SelectField).Help = "Own 3D shape, with its skin. Empty = the type's look"
		products.Fields.Add(&core.RelationField{Name: "compartment", MaxSelect: 1, CollectionId: comps.Id, Help: "Overrides the compartment recommended for its type. Empty = the type's"})
		if err := app.Save(products); err != nil {
			return err
		}

		recs, err := app.FindAllRecords(types)
		if err != nil {
			return err
		}
		for _, r := range recs {
			if l, ok := looks[r.GetString("key")]; ok {
				r.Set("shape", l[0])
				r.Set("skin", l[1])
			}
			r.Set("size", "m")
			if err := app.Save(r); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		types, err := app.FindCollectionByNameOrId("types")
		if err != nil {
			return err
		}
		for _, f := range []string{"shape", "skin", "size"} {
			types.Fields.RemoveByName(f)
		}
		if err := app.Save(types); err != nil {
			return err
		}
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		products.Fields.RemoveByName("compartment")
		products.Fields.GetByName("size").(*core.SelectField).Required = true
		return app.Save(products)
	})
}
