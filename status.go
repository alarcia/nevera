package main

import (
	"fmt"
	"time"
)

// Alert thresholds. The mockup's opinion, not food-safety advice: tune here.
const (
	urgentExpiryDays = 1   // expired, today or tomorrow
	soonExpiryDays   = 4   // 2 to 4 days left
	watchExpiryDays  = 7   // 5 to 7 days left
	urgentOpenFactor = 1.5 // open for >= 1.5x the recommended time
	watchOpenFactor  = 0.8 // open for >= 80% of it
)

// Level orders urgency: lower is more urgent.
type Level int

const (
	Urgent Level = iota
	Soon
	Watch
	None
)

// Andorra observes CET/CEST; "days left" is a calendar-date difference there.
var tz = func() *time.Location {
	l, err := time.LoadLocation("Europe/Andorra")
	if err != nil {
		panic(err) // time/tzdata is embedded, so this cannot happen
	}
	return l
}()

// today returns the calendar date in Andorra as a UTC-midnight time, the same
// shape calDate gives stored dates, so subtracting two of them counts days.
func today(now time.Time) time.Time {
	y, m, d := now.In(tz).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// calDate keeps the date the owner typed: PocketBase stores it as UTC midnight.
func calDate(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysBetween(from, to time.Time) int { return int(to.Sub(from).Hours() / 24) }

type status struct {
	Level     Level
	Chip      string // short label on the card
	ExpLong   string
	OpenLabel string
}

// itemStatus merges the expiry and the open-too-long alerts into one chip.
// A zero expires or opened means "none"; maxOpen 0 means "no limit".
func itemStatus(now, expires, opened time.Time, maxOpen int) status {
	t := today(now)
	st := status{Level: None, ExpLong: ui.NoDate + " de caducidad", OpenLabel: ui.Unopened}
	expLevel, expChip := None, ui.NoDate

	if !expires.IsZero() {
		e := calDate(expires)
		left := daysBetween(t, e)
		switch {
		case left < 0:
			expLevel, expChip = Urgent, "Caducado"
			st.ExpLong = fmt.Sprintf("Caducó el %s (hace %s)", dateLabel(e), daysLabel(-left))
		case left == 0:
			expLevel, expChip = Urgent, "Caduca hoy"
			st.ExpLong = "Hoy, " + dateLabel(e)
		case left == 1:
			expLevel, expChip = Urgent, "Caduca mañana"
			st.ExpLong = "Mañana, " + dateLabel(e)
		default:
			expLevel, expChip = expiryLevel(left), "Caduca "+dateLabel(e)
			st.ExpLong = fmt.Sprintf("%s (en %s)", dateLabel(e), daysLabel(left))
		}
	}

	openLevel, openChip := None, ""
	if !opened.IsZero() {
		o := calDate(opened)
		days := daysBetween(o, t)
		st.OpenLabel = fmt.Sprintf("Abierto desde el %s (%s)", dateLabel(o), daysLabel(days))
		openChip = "Abierto " + daysLabel(days)
		if maxOpen > 0 {
			switch f := float64(days) / float64(maxOpen); {
			case f >= urgentOpenFactor:
				openLevel = Urgent
			case days > maxOpen:
				openLevel = Soon
			case f >= watchOpenFactor:
				openLevel = Watch
			}
		}
	}

	st.Level, st.Chip = expLevel, expChip
	if openLevel < expLevel || (expires.IsZero() && !opened.IsZero()) {
		st.Level, st.Chip = openLevel, openChip
	}
	return st
}

func expiryLevel(left int) Level {
	switch {
	case left <= urgentExpiryDays:
		return Urgent
	case left <= soonExpiryDays:
		return Soon
	case left <= watchExpiryDays:
		return Watch
	}
	return None
}

// Fill weights per size, in "fill points". A compartment's capacity is in the
// same unit; the index is only a guide and never blocks anything.
var sizeWeight = map[string]int{"xs": 1, "s": 2, "m": 3, "l": 4, "xl": 16}

// fillPercent is 0 when the compartment has no capacity set (index off).
func fillPercent(points, capacity int) int {
	if capacity <= 0 {
		return 0
	}
	return points * 100 / capacity
}
