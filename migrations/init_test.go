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
}
