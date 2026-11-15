package main

import (
	"html/template"
	"log"
	"net/http"
	_ "time/tzdata" // the final image has no zoneinfo; Europe/Andorra must resolve anyway

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	_ "github.com/alarcia/nevera/migrations"
)

// ponytail: placeholder home until the fridge cutaway lands; proves the data path end to end.
var home = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="es"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>nevera</title>
<h1>nevera</h1>
<ul>{{range .}}<li>{{.GetString "name"}}</li>{{end}}</ul>
</html>`))

func main() {
	app := pocketbase.New()

	// Dashboard schema edits made during `go run` are written as Go migrations.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  osutils.IsProbablyGoRun(),
		Dir:          "migrations",
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/{$}", func(e *core.RequestEvent) error {
			comps, err := e.App.FindRecordsByFilter("compartments", "", "sort", 0, 0)
			if err != nil {
				return err
			}
			e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			e.Response.WriteHeader(http.StatusOK)
			return home.Execute(e.Response, comps)
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
