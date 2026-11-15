package main

import (
	"fmt"
	"time"
)

// ui holds every user-facing string (Spanish) so copy is reviewed in one place.
var ui = struct {
	App, Hint, Back, RecommendedFor, Empty                      string
	Type, Expiry, State, Quantity, Remaining, BetterIn          string
	Unopened, NoDate, Fill, FillOver, More                      string
	Urgent, Soon, Watch                                         string
	FridgeLabel, DetailLabel, RemainingLabel, NoRecommendations string
	Door, ShowDoor, HideDoor                                    string
}{
	ShowDoor:          "Mostrar puerta",
	HideDoor:          "Ocultar puerta",
	Door:              "PUERTA",
	App:               "nevera",
	Hint:              "Pulsa una balda o cajón para ver todo lo que hay",
	Back:              "← Volver a la nevera",
	RecommendedFor:    "Recomendada para",
	Empty:             "Aquí no hay nada todavía.",
	Type:              "Tipo",
	Expiry:            "Caducidad",
	State:             "Estado",
	Quantity:          "Cantidad",
	Remaining:         "Queda",
	BetterIn:          "Mejor en",
	Unopened:          "Sin abrir",
	NoDate:            "Sin fecha",
	Fill:              "Llenado",
	FillOver:          "lleno",
	More:              "más",
	Urgent:            "URGENTE",
	Soon:              "PRONTO",
	Watch:             "VIGILAR",
	FridgeLabel:       "Nevera",
	DetailLabel:       "Detalle de balda",
	RemainingLabel:    "Cantidad restante",
	NoRecommendations: "sin recomendación",
}

func daysLabel(n int) string {
	if n == 1 {
		return "1 día"
	}
	return fmt.Sprintf("%d días", n)
}

func countLabel(n int) string {
	if n == 1 {
		return "1 producto"
	}
	return fmt.Sprintf("%d productos", n)
}

var months = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

func dateLabel(t time.Time) string {
	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()-1])
}
