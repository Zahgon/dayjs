package dayjs

// The two en-US Intl.DateTimeFormat outputs the timezone plugin reads, as V8 and ICU produce
// them in Node.js 24: formatToParts with numeric year, two-digit month, day, hour (hour12 false,
// so hours 00 to 23), minute and second plus a zone name, and Date.prototype.toLocaleString
// ("M/d/y, h:mm:ss\u202fa"). The calendar is the proleptic Gregorian one and years are years of
// the era, so 5 BCE shows as 5.

import (
	"math"
	"strconv"

	"github.com/iamkun/dayjs-go/internal/icu"
)

type dtfPart struct {
	Type  string
	Value string
}

// resolveTimeZone validates an Intl timeZone option the way V8 does; undefined means the
// default (local) zone.
func resolveTimeZone(tz any) string {
	if isUndef(tz) {
		return localZone()
	}
	name := toJSString(tz)
	c, ok := icu.CanonicalTimeZone(name)
	if !ok {
		throw("RangeError", "Invalid time zone specified: "+name)
	}
	return c
}

// zoneFields are the wall-clock fields of the UTC time value t in a zone.
func zoneFields(t float64, zone string) dateFields {
	off := float64(icu.ZoneOffset(zone, t) * 1000)
	return fieldsOf(t + off)
}

func yearOfEra(y float64) string {
	if y <= 0 {
		y = 1 - y
	}
	return strconv.FormatInt(int64(y), 10)
}

func two(n float64) string {
	s := strconv.FormatInt(int64(n), 10)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

var timeZoneNameStyles = map[string]bool{
	"short": true, "long": true, "shortOffset": true, "longOffset": true,
	"shortGeneric": true, "longGeneric": true,
}

// formatToPartsEnUS is new Intl.DateTimeFormat('en-US', {hour12: false, timeZone, year:
// 'numeric', month, day, hour, minute, second: '2-digit', timeZoneName}).formatToParts(t).
func formatToPartsEnUS(t float64, timeZone any, timeZoneName string) []dtfPart {
	zone := resolveTimeZone(timeZone)
	if !timeZoneNameStyles[timeZoneName] {
		throw("RangeError", "Value "+timeZoneName+" out of range for Intl.DateTimeFormat options property timeZoneName")
	}
	if math.IsNaN(t) {
		throw("RangeError", "Invalid time value")
	}
	f := zoneFields(t, zone)
	return []dtfPart{
		{"month", two(f.month + 1)}, {"literal", "/"}, {"day", two(f.date)}, {"literal", "/"},
		{"year", yearOfEra(f.year)}, {"literal", ", "}, {"hour", two(f.hours)}, {"literal", ":"},
		{"minute", two(f.minutes)}, {"literal", ":"}, {"second", two(f.seconds)}, {"literal", " "},
		{"timeZoneName", zoneNameEnUS(zone, t, timeZoneName)},
	}
}

// zoneNameEnUS is the timeZoneName part of an en-US format.
func zoneNameEnUS(zone string, t float64, style string) string {
	off := icu.ZoneOffset(zone, t)
	switch style {
	case "short", "long":
		if name, ok := icu.ZoneName(zone, t, style == "long"); ok {
			return name
		}
		return gmtName(off, style == "long")
	case "shortOffset":
		return gmtName(off, false)
	case "longOffset":
		return gmtName(off, true)
	}
	throw("RangeError", "Unsupported timeZoneName "+style)
	return ""
}

// gmtName is ICU's localized GMT format for en-US.
func gmtName(off int64, long bool) string {
	if off == 0 {
		return "GMT"
	}
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	h, m, s := off/3600, off/60%60, off%60
	out := "GMT" + sign
	if long {
		out += two(float64(h)) + ":" + two(float64(m))
	} else {
		out += strconv.FormatInt(h, 10)
		if m != 0 || s != 0 {
			out += ":" + two(float64(m))
		}
	}
	if s != 0 {
		out += ":" + two(float64(s))
	}
	return out
}

// toLocaleStringEnUS is new Date(t).toLocaleString('en-US', {timeZone}).
func toLocaleStringEnUS(t float64, timeZone any) string {
	zone := resolveTimeZone(timeZone)
	if math.IsNaN(t) {
		return "Invalid Date"
	}
	f := zoneFields(t, zone)
	h := int64(f.hours) % 12
	if h == 0 {
		h = 12
	}
	ampm := "AM"
	if f.hours >= 12 {
		ampm = "PM"
	}
	return strconv.FormatInt(int64(f.month)+1, 10) + "/" + strconv.FormatInt(int64(f.date), 10) + "/" +
		yearOfEra(f.year) + ", " + strconv.FormatInt(h, 10) + ":" + two(f.minutes) + ":" + two(f.seconds) +
		"\u202f" + ampm
}

// resolvedTimeZone is Intl.DateTimeFormat().resolvedOptions().timeZone.
func resolvedTimeZone() string { return localZone() }
