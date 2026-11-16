package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
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
	adminTpl  = parse("templates/admin.html")
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
	funcs := template.FuncMap{"look": func(shape, skin string) slotView { return slotView{Shape: shape, Skin: skin} }}
	return template.Must(template.New("base.html").Funcs(funcs).ParseFS(assets, "templates/base.html", page))
}

type page struct {
	T      any
	Title  string
	V      string // asset version, see assetVer
	Body   string // body class; "fit" locks the page to the viewport height
	Fridge *fridgeView
	Comp   *compView
	Add    *addView
}

func render(e *core.RequestEvent, t *template.Template, p page) error {
	p.T, p.V = ui, assetVer
	add, err := addPage(e.App, time.Now())
	if err != nil {
		return err
	}
	p.Add = add
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", p); err != nil {
		return err
	}
	return e.HTML(http.StatusOK, buf.String())
}

// frag is one template of the admin screen with its data.
type frag struct {
	name string
	data any
}

// adminFragments renders pieces of the admin screen from fresh data, in order.
// A card that does not exist is a 404.
func adminFragments(e *core.RequestEvent, pick func(*adminView) []frag) error {
	v, err := adminPage(e.App)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for _, f := range pick(v) {
		if p, ok := f.data.(*adminProduct); ok && p == nil {
			return e.NotFoundError("", nil)
		} else if t, ok := f.data.(*adminType); ok && t == nil {
			return e.NotFoundError("", nil)
		}
		if err := adminTpl.ExecuteTemplate(&buf, f.name, f.data); err != nil {
			return err
		}
	}
	return e.HTML(http.StatusOK, buf.String())
}

func adminFragment(e *core.RequestEvent, name string, pick func(*adminView) any) error {
	return adminFragments(e, func(v *adminView) []frag { return []frag{{name, pick(v)}} })
}

// adminErr answers a failed admin edit with a message the screen shows as is.
func adminErr(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return e.NotFoundError("", err)
	case errors.Is(err, errHasItems):
		return e.Error(http.StatusConflict, ui.AdminHasItems, nil)
	}
	return e.BadRequestError(ui.BadForm, err)
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
		// Same-origin form posts only: there is no app login, the browser carries
		// the Cloudflare Access cookie, so a cross-site form must not ride on it.
		cop := http.NewCrossOriginProtection()
		post := func(path string, h func(e *core.RequestEvent, f url.Values) error) {
			se.Router.POST(path, func(e *core.RequestEvent) error {
				if err := cop.Check(e.Request); err != nil {
					return e.ForbiddenError("", err)
				}
				if err := e.Request.ParseForm(); err != nil {
					return e.BadRequestError(ui.BadForm, err)
				}
				return h(e, e.Request.PostForm)
			})
		}
		post("/items", func(e *core.RequestEvent, f url.Values) error {
			comp, err := createItem(e.App, f, time.Now())
			if err != nil {
				// ponytail: the form validates on the client too, so a bare error page
				// is rare; re-render the dialog with the input if it ever bites.
				return e.BadRequestError(ui.BadForm, err)
			}
			return e.Redirect(http.StatusSeeOther, "/?added="+url.QueryEscape(comp))
		})

		// The admin screen: the page fetches these as HTML fragments, no navigation.
		se.Router.GET("/admin", func(e *core.RequestEvent) error {
			return adminFragment(e, "admin", func(v *adminView) any { return v })
		})
		// Creating answers with the new card plus a fresh blank one to replace the
		// form that was used, in the order they sit on the screen.
		post("/admin/types", func(e *core.RequestEvent, f url.Values) error {
			id, err := createType(e.App, f)
			if err != nil {
				return adminErr(e, err)
			}
			return adminFragments(e, func(v *adminView) []frag {
				return []frag{{"typecard", v.NewType}, {"typecard", v.typeByID(id)}}
			})
		})
		post("/admin/products", func(e *core.RequestEvent, f url.Values) error {
			id, err := createProduct(e.App, f)
			if err != nil {
				return adminErr(e, err)
			}
			return adminFragments(e, func(v *adminView) []frag {
				return []frag{{"prodcard", v.NewProduct}, {"prodcard", v.productByID(id)}}
			})
		})
		post("/admin/types/{id}", func(e *core.RequestEvent, f url.Values) error {
			id := e.Request.PathValue("id")
			if err := saveType(e.App, id, f); err != nil {
				return adminErr(e, err)
			}
			return adminFragment(e, "typecard", func(v *adminView) any { return v.typeByID(id) })
		})
		post("/admin/products/{id}", func(e *core.RequestEvent, f url.Values) error {
			id := e.Request.PathValue("id")
			if err := saveProduct(e.App, id, f); err != nil {
				return adminErr(e, err)
			}
			return adminFragment(e, "prodcard", func(v *adminView) any { return v.productByID(id) })
		})
		post("/admin/products/{id}/delete", func(e *core.RequestEvent, f url.Values) error {
			if err := deleteProduct(e.App, e.Request.PathValue("id")); err != nil {
				return adminErr(e, err)
			}
			return e.NoContent(http.StatusNoContent)
		})
		items := func(path string, do func(app core.App, id string, f url.Values) (string, error)) {
			post(path, func(e *core.RequestEvent, f url.Values) error {
				pid, err := do(e.App, e.Request.PathValue("id"), f)
				if err != nil {
					return adminErr(e, err)
				}
				return adminFragment(e, "items", func(v *adminView) any { return v.productByID(pid) })
			})
		}
		items("/admin/items/{id}", saveItem)
		items("/admin/items/{id}/delete", func(app core.App, id string, _ url.Values) (string, error) { return removeItem(app, id) })
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
