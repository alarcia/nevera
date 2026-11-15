package main

import (
	"errors"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type sample struct {
	name, brand, typ, size, shape, skin, qty string
	exp, open                                *int // days from today (open: days ago); nil = none
	pct                                      int
}

func n(v int) *int { return &v }

// Invented products to see the app with data; placed in the first compartment
// recommended for their type, so they follow whatever layout is configured.
var samples = []sample{
	{"Lentejas con chorizo", "Casero", "leftovers", "m", "tupper", "tup1", "2 raciones", nil, n(4), 50},
	{"Pollo al curry", "Casero", "leftovers", "m", "tupper", "tup2", "1 ración", nil, n(1), 0},
	{"Boloñesa", "Casero", "leftovers", "m", "tupper", "tup3", "1 ración", nil, n(3), 30},
	{"Lasaña de verduras", "Cocina Lista", "ready_meal", "m", "tray", "lasagna", "400 g", n(6), nil, 0},
	{"Leche entera", "La Vaquería", "dairy", "l", "brick", "milk", "1 L", n(20), n(5), 35},
	{"Nata para cocinar", "La Vaquería", "dairy", "s", "brick", "cream", "200 ml", n(2), nil, 0},
	{"Yogur de fresa", "Valle Blanco", "dairy", "s", "cup", "yog", "1 ud", n(-1), nil, 0},
	{"Queso curado", "Sierra Alta", "cheese", "m", "block", "cheese", "250 g", n(40), n(12), 60},
	{"Jamón cocido", "Charcutería Roca", "cold_cuts", "s", "pack", "ham", "150 g", n(8), n(6), 40},
	{"Carne picada mixta", "Carnicería Pau", "meat", "m", "tray", "beef", "500 g", n(1), nil, 0},
	{"Filetes de merluza", "Mar Azul", "fish", "m", "tray", "hake", "300 g", n(0), nil, 0},
	{"Pollo asado entero", "Casero", "meat", "xl", "tray", "chicken", "1 ud", n(3), nil, 0},
	{"Lechuga romana", "Huerta Sol", "vegetable", "m", "leaf", "lettuce", "1 ud", n(4), nil, 0},
	{"Zanahorias", "Huerta Sol", "vegetable", "m", "long", "carrot", "6 uds", n(12), nil, 0},
	{"Manzanas Granny", "Frutas Mora", "fruit", "m", "round", "apple", "5 uds", n(15), nil, 0},
	{"Huevos camperos", "Granja El Roble", "eggs", "m", "carton", "egg", "6 de 12", n(10), n(8), 50},
	{"Mantequilla", "La Vaquería", "butter", "s", "block", "butter", "250 g", n(45), n(20), 55},
	{"Kétchup", "Don Tomate", "sauce", "s", "squeeze", "ketchup", "450 g", n(120), n(30), 25},
	{"Mayonesa", "Don Tomate", "sauce", "s", "squeeze", "mayo", "400 ml", n(60), n(70), 15},
	{"Pepinillos", "El Obrador", "preserve", "s", "jar", "pickle", "350 g", n(300), nil, 0},
	{"Agua mineral", "Font Clara", "drink", "l", "bottlel", "water", "1,5 L", nil, n(2), 80},
	{"Refresco de cola", "Kora", "drink", "l", "bottlel", "cola", "2 L", n(90), n(6), 20},
}

// devSeed refuses to touch a database that already has products.
func devSeed(app core.App, now time.Time) error {
	if c, err := app.CountRecords("products"); err != nil {
		return err
	} else if c > 0 {
		return errors.New("devseed: products already exist, refusing to add sample data")
	}
	t := today(now)
	at := func(days int) time.Time { return t.AddDate(0, 0, days) }

	types := map[string]*core.Record{}
	ts, err := app.FindAllRecords("types")
	if err != nil {
		return err
	}
	for _, r := range ts {
		types[r.GetString("key")] = r
	}
	comps, err := app.FindRecordsByFilter("compartments", "", "row,sort", 0, 0)
	if err != nil {
		return err
	}
	if len(comps) == 0 {
		return errors.New("devseed: no compartments configured")
	}
	home := func(typeID string) *core.Record {
		for _, c := range comps {
			for _, id := range c.GetStringSlice("types") {
				if id == typeID {
					return c
				}
			}
		}
		return comps[0]
	}

	pc, _ := app.FindCollectionByNameOrId("products")
	ic, _ := app.FindCollectionByNameOrId("items")
	for _, s := range samples {
		ty := types[s.typ]
		p := core.NewRecord(pc)
		p.Load(map[string]any{"name": s.name, "brand": s.brand, "type": ty.Id, "size": s.size, "shape": s.shape, "skin": s.skin})
		if err := app.Save(p); err != nil {
			return err
		}
		it := core.NewRecord(ic)
		it.Load(map[string]any{"product": p.Id, "compartment": home(ty.Id).Id, "quantity": s.qty, "remainingPct": s.pct})
		if s.exp != nil {
			it.Set("expiresOn", at(*s.exp))
		}
		if s.open != nil {
			it.Set("openedAt", at(-*s.open))
		}
		if err := app.Save(it); err != nil {
			return err
		}
	}
	return nil
}
