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

	// Add dialog.
	Add, AddTitle, NewProduct, Close, Change, Known string
	StepWhat, StepWhen, StepWhere                   string
	Name, Brand, BrandHint, Size, Shape, Colors     string
	Opened, QuantityHint, MoreFields                string
	Suggested, SavedIn, SavedNote, BadForm, NoSkin  string
	SaveAnother, Save                               string
	SugHome, SugFull, SugNoHome, SugOwn             string

	// Admin screen.
	Admin, AdminTitle, AdminBack, PocketBase          string
	AdminTypes, AdminProducts, AdminFilter, AdminNone string
	AdminLook, AdminDays, AdminShelf, AdminNoShelf    string
	AdminLikeType, AdminStored, AdminNothing          string
	AdminRemove, AdminDelete, AdminSure, AdminSaved   string
	AdminHasItems, AdminPick, AdminDone, Cancel       string
	AdminLoading, AdminFailed                         string
	AdminNoDate, AdminEdit, AdminNewType              string
	AdminNewProduct, AdminCreate, AdminPickType       string
	AdminLeave, AdminLess                             string
}{
	Add:          "añadir",
	AddTitle:     "Añadir a la nevera",
	NewProduct:   "Producto nuevo",
	Close:        "Cerrar",
	Change:       "Cambiar",
	Known:        "Ya lo tenías",
	StepWhat:     "¿Qué es?",
	StepWhen:     "¿Hasta cuándo?",
	StepWhere:    "¿Dónde va?",
	Name:         "Nombre",
	Brand:        "Marca",
	BrandHint:    "Casero, si lo has hecho tú",
	Size:         "Tamaño",
	Shape:        "Forma",
	Colors:       "Colores",
	Opened:       "Ya está abierto o empezado",
	QuantityHint: "1 L, 2 raciones, 6 de 12…",
	MoreFields:   "Más campos",
	Suggested:    "SUGERIDA",
	SavedIn:      "Guardado en",
	// {0} product, {1} compartment, {2} how many were added in this session.
	SavedNote:   "✓ {0} en {1} · llevas {2}",
	SaveAnother: "Guardar y añadir otro",
	Save:        "Guardar",
	BadForm:     "No se ha podido guardar: revisa los datos.",
	NoSkin:      "Colores por defecto",
	// Put-away reasons: %[1]s chosen compartment, %[2]s type, %[3]s recommended one.
	SugHome:   "Es el sitio pensado para %[2]s.",
	SugFull:   "%[3]s está llena; %[1]s es la que tiene más hueco.",
	SugNoHome: "Nada está pensado para %[2]s; %[1]s es la que tiene más hueco.",
	SugOwn:    "Es el sitio elegido para este producto.",

	Admin:         "admin",
	AdminTitle:    "Administración",
	AdminBack:     "← Volver",
	PocketBase:    "pocketbase",
	AdminTypes:    "Categorías",
	AdminProducts: "Productos",
	AdminFilter:   "Buscar…",
	AdminNone:     "Todavía no hay productos.",
	AdminLook:     "Aspecto",
	AdminDays:     "Días de caducidad",
	AdminShelf:    "Balda sugerida",
	AdminNoShelf:  "ninguna",
	AdminLikeType: "como la categoría",
	AdminStored:   "Guardado ahora",
	AdminNothing:  "Nada guardado.",
	AdminRemove:   "quitar",
	AdminDelete:   "Borrar producto",
	AdminSure:     "¿Seguro?",
	AdminSaved:    "Guardado ✓",
	AdminHasItems: "Aún hay artículos guardados de este producto: quítalos antes de borrarlo.",
	AdminPick:     "Elige el aspecto",
	AdminDone:     "Listo",
	Cancel:        "Cancelar",
	AdminLoading:  "Cargando…",
	AdminFailed:   "No se ha podido cargar la administración.",
	// Days: empty means "not set", 0 is the same day.
	AdminNoDate:     "sin fecha",
	AdminEdit:       "editar",
	AdminNewType:    "Nueva categoría",
	AdminNewProduct: "Nuevo producto",
	AdminCreate:     "Crear",
	AdminPickType:   "Elige una categoría",
	AdminLeave:      "¿Salir sin guardar?",
	AdminLess:       "menos",

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

// Sizes, smallest first (keys from the products.size field).
var sizeLabels = map[string]string{"xs": "Diminuto", "s": "Pequeño", "m": "Mediano", "l": "Grande", "xl": "Enorme"}

// Shape and skin names, only for screen readers: the pickers are visual.
var shapeLabels = map[string]string{
	"brick": "Brik", "bottle": "Botella", "bottlel": "Botella grande", "jar": "Tarro",
	"squeeze": "Bote blando", "tupper": "Táper", "tray": "Bandeja", "block": "Bloque",
	"can": "Lata", "carton": "Huevera", "cup": "Vasito", "pack": "Paquete plano",
	"leaf": "Hojas", "round": "Pieza redonda", "long": "Pieza alargada", "melon": "Melón",
}

var skinLabels = map[string]string{
	"milk": "Leche", "cream": "Nata", "ketchup": "Kétchup", "mayo": "Mayonesa", "mustard": "Mostaza",
	"bbq": "Barbacoa", "jam": "Mermelada", "pickle": "Encurtido", "pesto": "Pesto", "water": "Agua",
	"oj": "Zumo de naranja", "cola": "Refresco de cola", "beer": "Cerveza", "tup1": "Táper azul",
	"tup2": "Táper rojo", "tup3": "Táper verde", "chicken": "Pollo", "beef": "Ternera",
	"salmon": "Salmón", "hake": "Merluza", "lasagna": "Lasaña", "cheese": "Queso",
	"butter": "Mantequilla", "egg": "Huevos", "yog": "Yogur rosa", "yog2": "Yogur azul",
	"ham": "Jamón", "lettuce": "Lechuga", "tomato": "Tomate", "apple": "Manzana",
	"lemon": "Limón", "carrot": "Zanahoria", "zucchini": "Calabacín", "melon": "Melón",
}
