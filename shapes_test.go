package main

import (
	"math"
	"regexp"
	"strconv"
	"testing"
)

// Every shape in items.css carries --e (the most room it takes while it spins)
// and --lift (how far its cap rises above its box). The CSS uses them to keep
// each sprite inside its frame, so they must follow the dimensions on the same
// line: this recomputes them and says what to write after a dimension changes.
func TestShapeFit(t *testing.T) {
	b, err := assets.ReadFile("static/items.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)

	defaults := map[string]float64{"w": 22, "h": 30, "d": 18, "cw": 8, "ch": 5, "cd": 8} // .it
	noCap := map[string]bool{}
	for _, m := range regexp.MustCompile(`\.k-(\w+) \.cap`).FindAllStringSubmatch(css, -1) {
		noCap[m[1]] = true
	}
	shapes := regexp.MustCompile(`(?m)^\.k-(\w+)\{([^}]*)\}$`).FindAllStringSubmatch(css, -1)
	if len(shapes) != 16 {
		t.Fatalf("%d shapes in items.css, want 16", len(shapes))
	}

	tilt := 14 * math.Pi / 180 // the rotateX every sprite is drawn with
	for _, m := range shapes {
		v := map[string]float64{}
		for k, x := range defaults {
			v[k] = x
		}
		e, lift := -1.0, -1.0
		for _, kv := range regexp.MustCompile(`--(\w+):(\d+)`).FindAllStringSubmatch(m[2], -1) {
			n, _ := strconv.ParseFloat(kv[2], 64)
			switch kv[1] {
			case "e":
				e = n
			case "lift":
				lift = n
			default:
				v[kv[1]] = n
			}
		}
		diag := math.Hypot(v["w"], v["d"]) // the box turning
		wide, rise := diag, 0.0
		if !noCap[m[1]] {
			wide, rise = math.Max(diag, math.Hypot(v["cw"], v["cd"])), v["ch"]
		}
		tall := (v["h"]+rise)*math.Cos(tilt) + diag*math.Sin(tilt)
		if want := math.Ceil(math.Max(wide, tall)); e != want || lift != rise {
			t.Errorf(".k-%s has --e:%v --lift:%v, want --e:%v --lift:%v", m[1], e, lift, want, rise)
		}
	}
}
