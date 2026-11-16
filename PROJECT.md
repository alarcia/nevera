# nevera: project brief

Hand this file to the implementing agent together with the mockup (`Main.dc.html`). It records what the product is, what the mockup already decides, and what is still open. Where something is marked **suggestion** or **open**, ask the owner before locking it in.

## 1. What it is

`nevera` is a self-hosted web app (mobile first) to keep track of what is inside the household fridge: what it is, where it is, when it expires, whether it is opened, and how much is left. The name is Spanish ("fridge"); everything else (code, docs, commits, repo description) is in English. The **UI copy is in Spanish** (see section 9).

Two people use it, daily, mostly from a phone. It will be deployed at `nevera.alarcia.dev` on the owner's own server. The repository is **public**: no secrets, no personal data in seed data, ship a `.env.example`.

### Scope

- **v1: the fridge only.** Shelves, drawers and door compartments.
- **Later (not v1): freezer and pantry/cupboards.** Do not build them, but do not paint the data model into a corner (section 4).
- Not in v1: shopping lists, barcode scanning, recipes, multiple households, public sign-up.

## 2. The four screens (all present in the mockup)

1. **Fridge cross-section** (home). A side-on cut of the fridge: shelves stacked vertically, two drawers at the bottom, and the open door on the right with its own small shelves. Products are drawn piled up and overlapping, as when you open a real fridge. Each compartment shows its name, what it is recommended for, and its product count. Tapping a compartment opens screen 2.
2. **Shelf detail.** A larger view of one compartment with every product in an orderly grid. Hovering (or keyboard-focusing) a product enlarges the card, spins the 3D item, and reveals: type, expiry, opened-since, quantity, remaining % bar, and a "better in X" hint if it is not in its recommended compartment.
3. **Put away the groceries.** Toggled from the header. **Changed from the mockup: there is no pre-built shopping list.** The user creates items one by one as they unpack (picking a known product from the catalog, or a new one), entering the expiry date, which is pre-filled with today + the product's usual shelf life. For each item, the app suggests a compartment (by type, size and how full each compartment is) and explains why; the compartment is highlighted in the fridge. The user confirms with one button ("Colocado en X") **or** taps any other compartment to say where it really went. Later (not MVP): scan a barcode to identify the product, so only the expiry date is left to confirm; and suggest reorganising (move X down to make room).
4. **Alerts.** A side panel listing items near expiry or open for too long, in three urgency levels, most urgent first. Tapping an alert jumps to that item's compartment.

## 3. Domain model

### Product

| Field | Notes |
|---|---|
| `name` | e.g. "Leche entera" |
| `brand` | free text; homemade leftovers use "Casero" |
| `type` | enum, see below |
| `expiresOn` | date or null (leftovers and water often have none) |
| `dateKind` | **suggestion**: `use_by` (safety, "consumir antes de") vs `best_before` ("consumo preferente"). Alerts for `use_by` should weigh more. Not in the mockup |
| `opened` + `openedAt` | boolean plus date. "Opened" also means "started" (a leftover tupper, a cheese block) |
| `maxDaysOpen` | recommended days to finish it once opened. Mockup values are invented; derive a default per type/product and let the user override. Do not present these as food-safety advice |
| `remainingPct` | 0 to 100, only for things that deplete (sauces, liquids, jars, partly used items); null otherwise |
| `quantity` | free text in the mockup ("2 raciones", "1,5 L", "6 de 12"). **Open**: keep free text or structure it (amount + unit) |
| `size` | Every product has one. Rough bulk, not weight: diminuto (garlic clove), pequeño (yogurt), mediano (apple), grande (1 L carton), enorme (whole roast chicken; 2 or 3 per shelf). Drives each compartment's fill index and the put-away suggestion |
| `compartmentId` | where it is now |
| `shape`, `skin` | visual only: which 3D shape and colour scheme draw it (section 6) |
| `addedAt` | not in the mockup, but useful (history, "has been here for N days") |

**Suggestion:** split a reusable *catalog item* (name, brand, type, size, shape, skin, default `maxDaysOpen`) from the concrete *stock item* (expiry, opened, remaining, location). Re-buying milk should be one tap, and this is also where a barcode lookup would plug in later.

### Types used in the mockup

Sobras, Plato preparado, Verdura, Fruta, Lácteo, Queso, Embutido, Salsa, Conserva, Carne, Pescado, Huevos, Mantequilla, Bebida. The brief also mentions "ingrediente", which the mockup does not use. Keep types as an extensible set with stable English keys and Spanish labels.

### Compartment

`id`, `kind` (`shelf` | `drawer` | `door`), `name`, `recommendedTypes[]`, `maxSizeRank` (S=1 … XL=4; how big an item fits), `why` (short explanation shown to the user), and an order or position.

- **Any product can go in any compartment.** The recommendation is a hint, never a constraint.
- **No maximum number of items per compartment.**
- Drawers and door shelves are smaller than main shelves, hence the lower `maxSizeRank`.

Mockup seed (placeholder; **the real fridge layout should be configurable or seeded to match the owner's actual fridge, so ask**):

| Compartment | Kind | Recommended | Max size |
|---|---|---|---|
| Balda alta | shelf | Sobras, Plato preparado | 4 |
| Balda media | shelf | Lácteo, Queso, Embutido | 4 |
| Balda baja | shelf | Carne, Pescado | 4 |
| Cajón izq. | drawer | Verdura | 3 |
| Cajón dcho. | drawer | Fruta | 3 |
| Puerta alta | door | Huevos, Mantequilla | 2 |
| Puerta media | door | Salsa, Conserva | 2 |
| Puerta baja | door | Bebida | 3 |

### Future-proofing for freezer and pantry

Model a generic **storage unit → compartments → products** hierarchy, with the fridge as the first unit. Expect these differences later, so avoid hard-coding fridge assumptions:

- Freezer: shelf life measured in months, a "frozen on" date, possibly thaw tracking.
- Pantry: mostly unopened shelf-stable goods, quantity matters more than expiry.

The home screen currently draws the fridge cutaway directly; a unit switcher would sit above it.

## 4. Behaviour the mockup implements (use it as the reference)

### Alert levels

An item can raise two independent alerts (expiry and open-too-long). Sort by level, then by days left.

| Level | Expiry rule (days left) | Open rule (days open vs `maxDaysOpen`) |
|---|---|---|
| URGENTE | expired, today, or tomorrow | open ≥ 1.5× the recommended time |
| PRONTO | 2 to 4 days | open > recommended time |
| VIGILAR | 5 to 7 days | open ≥ 80% of recommended time |

Thresholds are the mockup's opinion; make them configurable constants, not magic numbers scattered around.

### Put-away suggestion (mockup heuristic, crude, improve it)

1. Find the first compartment whose `recommendedTypes` contains the item's type (fall back to the middle shelf).
2. If the item's size rank fits that compartment's `maxSizeRank`, suggest it.
3. Otherwise: XL goes to the top shelf (most headroom); anything else too big for its compartment goes to the lowest door shelf (tall bottles stand upright).
4. Always return a human-readable reason, shown to the user.

On confirmation the item becomes a stock item in the chosen compartment, with `opened = false`.

## 5. Not designed yet (the mockup has no UI for these)

Needed for a usable v1, and the agent should propose flows before building:

- Add a product manually (and quickly re-add a known one).
- Edit a product; mark it opened (sets `openedAt`); update remaining %.
- Mark as consumed or discard (consider a short history, which later enables "what we throw away" insights).
- Move a product between compartments outside put-away mode.
- Build the "new shopping" list that feeds put-away mode (the mockup hard-codes 9 items).
- Configure compartments to match the real fridge.
- Empty states, error states, and loading states.

## 6. Visual direction

- **Look:** retro PSX-era game inventory. Products are low-poly 3D items with dithered textures, drawn at pixel-art scale; the active (hovered or focused) item rotates in stepped frames, like a pre-rendered sprite. Dark UI chrome around a bright fridge interior.
- **Fonts:** Jersey 15 for everything (headings, labels, text): a pixel font whose C, O and 0 can be told apart, unlike Pixelify Sans. `--px` and `--ui` in `static/app.css` name the font for each role, and `--fs-s/m/l/xl` are the four text sizes (labels and notes, text you read and tap, headings, logo). The font draws small for its size, so the sizes are a step up from a normal font's: text is 20px, labels 15px.
- **How the mockup draws an item:** a small CSS 3D cuboid (5 faces) plus an optional cap (another small cuboid, for lids and nozzles). A `shape` class sets the dimensions (brick, bottle, bottle-large, jar, squeeze, tupper, tray, block, can, carton, cup, pack, leaf, round, long, melon) and a `skin` class sets the colours (milk, ketchup, tupper variants, meat tray colours, and so on). Dither comes from a repeating conic gradient; the spin is a CSS keyframe with `steps(36)`.
- **Depth in the cutaway:** items overlap with negative margins, and every third item is pushed back (shifted up, scaled down, darkened). Drawers and door shelves get a translucent front panel that hides the lower part of the items.
- **Performance warning:** there is no cap on items per compartment, and every item is ~10 DOM nodes with 3D transforms. Fine for 30 products; not for hundreds. In the cutaway, render only a bounded number per compartment (the rest implied), and virtualise or lazy-render the shelf grid. If it still struggles, move to pre-rendered sprites or a small canvas/WebGL scene with nearest-neighbour filtering. **Open:** decide with the owner whether to keep pure CSS 3D.
- **Accessibility (keep it):** real `<button>`s for compartments; hover details also appear on keyboard focus; `prefers-reduced-motion` disables spin and scaling; urgency never relies on colour alone (each level has a text label); contrast ≥ 4.5:1 for text.
- **Responsive:** the layout wraps into columns on narrow screens. Verify the cutaway and hover cards on a real phone. Hover does not exist on touch, so the detail card needs a tap/long-press equivalent. This is **not solved in the mockup**.

## 7. About the mockup file

- `Main.dc.html` is written in a design-canvas component format (`<x-dc>`, `{{holes}}`, `<sc-for>`, a `DCLogic` class, plus a `support.js` runtime). **It will not run as a plain HTML page and is not production code.** Treat it as a visual and behavioural reference.
- Reusable almost verbatim: the CSS for the 3D items (shapes, skins, dither, spin), the colour and typographic choices, the seed data, and the logic in `zones()`, `recommend()`, `expInfo()` and `openInfo()`.
- The mockup's state is in-memory only, with "today" pinned to **13 Nov 2026** and invented products, brands and open-time limits. Real code must use the real date and the owner's real data.

## 8. Open decisions: ask the owner, one at a time

**Decided (Nov 2026):** stack is Go + PocketBase (SQLite, its dashboard for data management) with server-rendered templates and htmx; one container on the owner's Raspberry Pi, same pattern as their other apps; no app login, access through Cloudflare Access with an allowlist; no offline support; MIT licence; compartments and their recommended types are user-configurable; freezer and pantry come later as more storage units.

1. **Stack.** Frontend framework, backend language, and whether this is one app or API plus client. The owner's other projects use Astro, Python, TypeScript, PHP and Java, but no stack is chosen here.
2. **Storage.** Likely a small embedded or self-hosted database (SQLite is the obvious candidate for two users; confirm).
3. **Auth.** Two household users on a public domain. Options: shared login, per-user accounts, passkeys, or auth at the reverse proxy.
4. **Shared state.** Both phones see the same fridge. Do changes need to appear live, or is refresh-on-open enough?
5. **PWA.** Installable on the phone's home screen, since it is used daily. Offline support or not?
6. **Notifications.** Surface alerts only inside the app, or push them (web push, Telegram, email)? The owner already has a project that sends Telegram and email alerts; ask before assuming any reuse.
7. **Deployment.** The owner's server, container or not, reverse proxy, backups. Ask about the server before writing deploy files.
8. **Data entry.** Manual only at first, or a product lookup (e.g. an open food database) once barcodes are in scope?
9. **Licence.** Repository is public; ask which licence.
10. **Time handling.** The owner is in Europe/Andorra. Compute "days left" from calendar dates in that time zone, not from timestamps.

## 9. Working agreement

- The UI is in Spanish. Keep all user-facing strings in one place so they are easy to review and change.
- Code, comments, commits, issues and docs are in English.
- Work in small steps, starting with the data model and the fridge cutaway on real data. Show something working early.
- Ask before making a decision listed in section 8, or anything that is hard to change later.
- Keep secrets out of the repo. The repository is public.
