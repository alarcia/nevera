package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"
	_ "time/tzdata" // the final image has no zoneinfo; Europe/Andorra must resolve anyway

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"github.com/spf13/cobra"

	_ "github.com/alarcia/nevera/migrations"
)

//go:embed templates static
var assets embed.FS

var (
	fridgeTpl = parse("templates/fridge.html")
	compTpl   = parse("templates/compartment.html")
)

// assetVer changes whenever an embedded static file does. It goes in the asset
// URLs so a CDN or browser can never serve a stale stylesheet with fresh HTML.
var assetVer = func() string {
	h := sha256.New()
	fs.WalkDir(assets, "static", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := assets.ReadFile(p)
			h.Write([]byte(p))
			h.Write(b)
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:10]
}()

func parse(page string) *template.Template {
	return template.Must(template.ParseFS(assets, "templates/base.html", page))
}

type page struct {
	T      any
	Title  string
	V      string // asset version, see assetVer
	Body   string // body class; "fit" locks the page to the viewport height
	Fridge *fridgeView
	Comp   *compView
}

func render(e *core.RequestEvent, t *template.Template, p page) error {
	p.T, p.V = ui, assetVer
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", p); err != nil {
		return err
	}
	return e.HTML(http.StatusOK, buf.String())
}

func main() {
	app := pocketbase.New()

	// Dashboard schema edits made during `go run` are written as Go migrations.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  osutils.IsProbablyGoRun(),
		Dir:          "migrations",
	})

	// Development only: fills a local database with invented products. Never run in production.
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "devseed",
		Short: "Fill an empty local database with invented products (development only)",
		RunE: func(*cobra.Command, []string) error {
			if err := app.RunAllMigrations(); err != nil {
				return err
			}
			return devSeed(app, time.Now())
		},
	})

	static, _ := fs.Sub(assets, "static")
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/static/{path...}", apis.Static(static, false))

		se.Router.GET("/{$}", func(e *core.RequestEvent) error {
			v, err := fridgePage(e.App, time.Now())
			if errors.Is(err, errNotFound) {
				return e.NotFoundError("", nil)
			} else if err != nil {
				return err
			}
			return render(e, fridgeTpl, page{Title: ui.App, Body: "fit", Fridge: v})
		})

		se.Router.GET("/c/{id}", func(e *core.RequestEvent) error {
			v, err := compartmentPage(e.App, e.Request.PathValue("id"), time.Now())
			if errors.Is(err, errNotFound) {
				return e.NotFoundError("", nil)
			} else if err != nil {
				return err
			}
			return render(e, compTpl, page{Title: v.Name + " · " + ui.App, Comp: v})
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
