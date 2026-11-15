// Package migrations holds the schema and the seed of the default fridge.
//
// Model: unit (fridge, later freezer/pantry) → compartments → items.
// An item is one concrete thing stored somewhere; its product is the reusable
// catalog entry (re-buying milk is one tap; a barcode lookup plugs in here later).
//
// PocketBase fields are never null: a number left empty is 0 and a date is "".
// Where that matters the field's Help says what 0 or "" means.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Sizes, smallest first. Keys are stable; labels live in the UI strings.
// diminuto (a garlic clove) · pequeño (a yogurt) · mediano (an apple)
// grande (a 1 L milk carton) · enorme (a whole roast chicken)
var sizes = []string{"xs", "s", "m", "l", "xl"}

var shapes = []string{"brick", "bottle", "bottlel", "jar", "squeeze", "tupper", "tray", "block", "can", "carton", "cup", "pack", "leaf", "round", "long", "melon"}

var skins = []string{"milk", "cream", "ketchup", "mayo", "mustard", "bbq", "jam", "pickle", "pesto", "water", "oj", "cola", "beer", "tup1", "tup2", "tup3", "chicken", "beef", "salmon", "hake", "lasagna", "cheese", "butter", "egg", "yog", "yog2", "ham", "lettuce", "tomato", "apple", "lemon", "carrot", "zucchini", "melon"}

func init() {
	m.Register(func(app core.App) error {
		units := core.NewBaseCollection("units")
		units.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.SelectField{Name: "kind", Required: true, MaxSelect: 1, Values: []string{"fridge", "freezer", "pantry"}},
			&core.NumberField{Name: "sort", OnlyInt: true},
		)
		if err := app.Save(units); err != nil {
			return err
		}

		types := core.NewBaseCollection("types")
		types.Fields.Add(
			&core.TextField{Name: "key", Required: true, Help: "Stable English key, never shown"},
			&core.TextField{Name: "label", Required: true, Presentable: true},
			&core.NumberField{Name: "shelfLifeDays", OnlyInt: true, Help: "Suggested expiry = today + this. 0 = no expiry date"},
			&core.NumberField{Name: "maxDaysOpen", OnlyInt: true, Help: "Rough days to finish it once opened. 0 = no limit. Not food-safety advice"},
			&core.BoolField{Name: "startsOpen", Help: "Counts as opened from the moment it is stored (leftovers)"},
			&core.NumberField{Name: "sort", OnlyInt: true},
		)
		types.AddIndex("idx_types_key", true, "key", "")
		if err := app.Save(types); err != nil {
			return err
		}

		compartments := core.NewBaseCollection("compartments")
		compartments.Fields.Add(
			&core.RelationField{Name: "unit", Required: true, MaxSelect: 1, CollectionId: units.Id, CascadeDelete: true},
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.SelectField{Name: "kind", Required: true, MaxSelect: 1, Values: []string{"shelf", "drawer", "door"}},
			&core.RelationField{Name: "types", MaxSelect: 99, CollectionId: types.Id, Help: "Recommended types. A hint, never a constraint"},
			&core.TextField{Name: "why", Help: "Why those types go here, shown to the user"},
			&core.SelectField{Name: "maxSize", Required: true, MaxSelect: 1, Values: sizes, Help: "Biggest size that physically fits"},
			&core.NumberField{Name: "capacity", OnlyInt: true, Help: "Fill points it holds; sizes weigh xs=1 s=2 m=3 l=4 xl=16. Never a hard limit"},
			&core.NumberField{Name: "sort", OnlyInt: true},
		)
		if err := app.Save(compartments); err != nil {
			return err
		}

		products := core.NewBaseCollection("products")
		products.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.TextField{Name: "brand", Help: "Homemade leftovers use \"Casero\""},
			&core.RelationField{Name: "type", Required: true, MaxSelect: 1, CollectionId: types.Id},
			&core.SelectField{Name: "size", Required: true, MaxSelect: 1, Values: sizes},
			&core.SelectField{Name: "shape", MaxSelect: 1, Values: shapes, Help: "3D shape it is drawn with"},
			&core.SelectField{Name: "skin", MaxSelect: 1, Values: skins, Help: "Colours it is drawn with"},
			&core.NumberField{Name: "shelfLifeDays", OnlyInt: true, Help: "Overrides the type's. 0 = use the type's"},
			&core.NumberField{Name: "maxDaysOpen", OnlyInt: true, Help: "Overrides the type's. 0 = use the type's"},
		)
		if err := app.Save(products); err != nil {
			return err
		}

		items := core.NewBaseCollection("items")
		items.Fields.Add(
			&core.RelationField{Name: "product", Required: true, MaxSelect: 1, CollectionId: products.Id},
			&core.RelationField{Name: "compartment", Required: true, MaxSelect: 1, CollectionId: compartments.Id},
			&core.DateField{Name: "expiresOn", Help: "Calendar date, Europe/Andorra. Empty = no expiry"},
			&core.DateField{Name: "openedAt", Help: "Empty = unopened"},
			&core.NumberField{Name: "remainingPct", OnlyInt: true, Min: ptr(0), Max: ptr(100), Help: "Only for things that deplete. 0 = not tracked"},
			&core.TextField{Name: "quantity", Help: "Free text: \"2 raciones\", \"1,5 L\""},
			&core.DateField{Name: "removedAt", Help: "Empty = still stored. Set = history"},
			&core.SelectField{Name: "outcome", MaxSelect: 1, Values: []string{"consumed", "discarded"}},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		items.AddIndex("idx_items_compartment", false, "compartment, removedAt", "")
		if err := app.Save(items); err != nil {
			return err
		}

		return seed(app, units, types, compartments)
	}, func(app core.App) error {
		for _, name := range []string{"items", "products", "compartments", "types", "units"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}

func ptr(f float64) *float64 { return &f }

// seed creates the mockup's fridge layout and types. Placeholder values, all
// editable from the dashboard; no products or items (real data only).
func seed(app core.App, unitsC, typesC, compsC *core.Collection) error {
	unit := core.NewRecord(unitsC)
	unit.Load(map[string]any{"name": "Nevera", "kind": "fridge", "sort": 1})
	if err := app.Save(unit); err != nil {
		return err
	}

	// key, label, shelfLifeDays, maxDaysOpen, startsOpen. Rough defaults to edit.
	typeRows := []struct {
		key, label string
		life, open int
		startsOpen bool
	}{
		{"leftovers", "Sobras", 0, 3, true},
		{"ready_meal", "Plato preparado", 4, 2, false},
		{"vegetable", "Verdura", 7, 4, false},
		{"fruit", "Fruta", 10, 4, false},
		{"dairy", "Lácteo", 7, 4, false},
		{"cheese", "Queso", 30, 21, false},
		{"cold_cuts", "Embutido", 10, 5, false},
		{"sauce", "Salsa", 180, 60, false},
		{"preserve", "Conserva", 365, 30, false},
		{"meat", "Carne", 3, 1, false},
		{"fish", "Pescado", 2, 1, false},
		{"eggs", "Huevos", 21, 0, false},
		{"butter", "Mantequilla", 60, 30, false},
		{"drink", "Bebida", 90, 5, false},
	}
	typeID := map[string]string{}
	for i, t := range typeRows {
		r := core.NewRecord(typesC)
		r.Load(map[string]any{"key": t.key, "label": t.label, "shelfLifeDays": t.life, "maxDaysOpen": t.open, "startsOpen": t.startsOpen, "sort": i + 1})
		if err := app.Save(r); err != nil {
			return err
		}
		typeID[t.key] = r.Id
	}

	compRows := []struct {
		name, kind string
		types      []string
		maxSize    string
		capacity   int
		why        string
	}{
		{"Balda alta", "shelf", []string{"leftovers", "ready_meal"}, "xl", 48, "Temperatura estable y lejos de los crudos: lo ya cocinado se conserva mejor arriba."},
		{"Balda media", "shelf", []string{"dairy", "cheese", "cold_cuts"}, "xl", 48, "Frío medio y constante, el punto ideal para lácteos, queso y embutido."},
		{"Balda baja", "shelf", []string{"meat", "fish"}, "xl", 48, "Es la zona más fría, justo encima de los cajones: la mejor para carne y pescado crudos, y si gotean no manchan nada."},
		{"Cajón izq.", "drawer", []string{"vegetable"}, "l", 24, "El cajón conserva la humedad y evita que la verdura se seque."},
		{"Cajón dcho.", "drawer", []string{"fruit"}, "l", 24, "Separada de la verdura, la fruta madura más despacio y no la estropea."},
		{"Puerta alta", "door", []string{"eggs", "butter"}, "m", 12, "La puerta es la zona menos fría: suficiente para huevos y mantequilla."},
		{"Puerta media", "door", []string{"sauce", "preserve"}, "m", 16, "Salsas y conservas abiertas aguantan bien los cambios de temperatura de la puerta."},
		{"Puerta baja", "door", []string{"drink"}, "l", 20, "El hueco más alto de la puerta: las botellas caben de pie."},
	}
	for i, c := range compRows {
		ids := make([]string, len(c.types))
		for j, k := range c.types {
			ids[j] = typeID[k]
		}
		r := core.NewRecord(compsC)
		r.Load(map[string]any{"unit": unit.Id, "name": c.name, "kind": c.kind, "types": ids, "maxSize": c.maxSize, "capacity": c.capacity, "why": c.why, "sort": i + 1})
		if err := app.Save(r); err != nil {
			return err
		}
	}
	return nil
}
