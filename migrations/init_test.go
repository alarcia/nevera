package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// Applies every migration to an empty database and checks the seed landed.
func TestInitMigration(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer app.ResetBootstrapState()
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]int64{"units": 1, "types": 14, "compartments": 8} {
		got, err := app.CountRecords(name)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: got %d records, want %d", name, got, want)
		}
	}

	// Every type is drawn by default; a product may leave all of it to its type.
	types, err := app.FindAllRecords("types")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range types {
		if r.GetString("shape") == "" || r.GetString("size") == "" {
			t.Errorf("type %s has no default look or size", r.GetString("key"))
		}
	}
	products, err := app.FindCollectionByNameOrId("products")
	if err != nil {
		t.Fatal(err)
	}
	// Shelf life: 0 is the same day now, so what was "0 = none" reads -1.
	for _, r := range types {
		life := r.GetInt("shelfLifeDays")
		if (r.GetString("key") == "leftovers") != (life == -1) || life == 0 {
			t.Errorf("type %s has shelf life %d", r.GetString("key"), life)
		}
	}
	for _, c := range []string{"types", "products"} {
		col, _ := app.FindCollectionByNameOrId(c)
		if min := col.Fields.GetByName("shelfLifeDays").(*core.NumberField).Min; min == nil || *min != -1 {
			t.Errorf("%s.shelfLifeDays must allow -1", c)
		}
	}
	if products.Fields.GetByName("size").(*core.SelectField).Required || products.Fields.GetByName("compartment") == nil {
		t.Error("products must be able to inherit their size and have an own compartment")
	}
}
