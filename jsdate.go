package dayjs

// The JavaScript Date object as V8 implements it in Node.js 24: time values are float64 epoch
// milliseconds (NaN when invalid, outside ±8.64e15), local time comes from the zone named by the
// TZ environment variable through ICU's tables, and the constructor, Date.UTC, the getters and
// setters, getTimezoneOffset, toISOString, toUTCString and toString follow V8's builtins.

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/iamkun/dayjs-go/internal/icu"
)

const (
	msPerDay    = 86400000.0
	msPerHour   = 3600000.0
	msPerMinute = 60000.0
	msPerSecond = 1000.0
	maxTimeMs   = 8.64e15
	// maxTimeBeforeUTCMs is V8's bound on a local time value before it is converted to UTC.
	maxTimeBeforeUTCMs = maxTimeMs + 30*msPerDay
)

// toIntegerOrInfinity is the abstract operation of that name (-0 becomes +0).
func toIntegerOrInfinity(x float64) float64 {
	if math.IsNaN(x) {
		return 0
	}
	if math.IsInf(x, 0) {
		return x
	}
	t := math.Trunc(x)
	if t == 0 {
		return 0
	}
	return t
}

var dayFromMonth = [2][12]int64{
	{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334},
	{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335},
}

// makeDay is V8's MakeDay over arguments already passed through ToNumber.
func makeDay(year, month, date float64) float64 {
	if !(-1000000 <= year && year <= 1000000 && -10000000 <= month && month <= 10000000) ||
		math.IsNaN(date) || math.IsInf(date, 0) {
		return math.NaN()
	}
	y, m := int64(year), int64(month)
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	const yearDelta = 399999
	base := 365*(1970+yearDelta) + (1970+yearDelta)/4 - (1970+yearDelta)/100 + (1970+yearDelta)/400
	day := 365*(y+yearDelta) + (y+yearDelta)/4 - (y+yearDelta)/100 + (y+yearDelta)/400 - int64(base)
	leap := 0
	if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
		leap = 1
	}
	day += dayFromMonth[leap][m]
	return float64(day-1) + toIntegerOrInfinity(date)
}

// makeTime is MakeTime.
func makeTime(h, m, s, ms float64) float64 {
	for _, v := range []float64{h, m, s, ms} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return math.NaN()
		}
	}
	return toIntegerOrInfinity(h)*msPerHour + toIntegerOrInfinity(m)*msPerMinute +
		toIntegerOrInfinity(s)*msPerSecond + toIntegerOrInfinity(ms)
}

// makeDate is MakeDate.
func makeDate(day, time float64) float64 {
	if math.IsNaN(day) || math.IsInf(day, 0) || math.IsNaN(time) || math.IsInf(time, 0) {
		return math.NaN()
	}
	return time + day*msPerDay
}

// timeClip is TimeClip.
func timeClip(t float64) float64 {
	if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > maxTimeMs {
		return math.NaN()
	}
	return toIntegerOrInfinity(t)
}

// fullYear applies the constructor's and Date.UTC's rule that years 0 to 99 mean 1900 to 1999.
func fullYear(year float64) float64 {
	if !math.IsNaN(year) {
		if yi := toIntegerOrInfinity(year); yi >= 0 && yi <= 99 {
			return 1900 + yi
		}
	}
	return year
}

// dateArgs reads the (year, month[, date, hours, minutes, seconds, ms]) arguments of the
// constructor and Date.UTC and returns the time value they describe, before UTC conversion.
func dateArgs(args []any) float64 {
	n := [7]float64{math.NaN(), 0, 1, 0, 0, 0, 0}
	for i := 0; i < len(args) && i < 7; i++ {
		n[i] = toNumberStrict(args[i])
	}
	return makeDate(makeDay(fullYear(n[0]), n[1], n[2]), makeTime(n[3], n[4], n[5], n[6]))
}

// toNumberStrict is ToNumber, throwing for a BigInt as JavaScript does.
func toNumberStrict(v any) float64 {
	if _, ok := v.(*big.Int); ok {
		throw("TypeError", "Cannot convert a BigInt value to a number")
	}
	return toNumber(v)
}

// dateUTC is Date.UTC(...args).
func dateUTC(args ...any) float64 {
	if len(args) == 0 {
		return math.NaN()
	}
	return timeClip(dateArgs(args))
}

// ---------------------------------------------------------------------------------------------
// the local time zone

var (
	localOnce sync.Once
	localID   string
)

// localZone is the canonical identifier of the zone named by TZ (UTC when TZ is unset or not a
// zone ICU knows).
func localZone() string {
	localOnce.Do(func() {
		tz := strings.TrimPrefix(os.Getenv("TZ"), ":")
		if c, ok := icu.CanonicalTimeZone(tz); ok && tz != "" {
			localID = c
		} else {
			localID = "UTC"
		}
	})
	return localID
}

// localOffset is the offset in milliseconds that V8's ToLocal adds to the UTC time value t.
func localOffset(t float64) float64 {
	return float64(icu.ZoneOffset(localZone(), t) * 1000)
}

// toLocal is V8's DateCache::ToLocal.
func toLocal(t float64) float64 { return t + localOffset(t) }

// localToUTC converts a local time value to a UTC one as V8's SetLocalDateValue and the
// constructor do: out of range values are NaN, a skipped or repeated wall time takes the offset
// in effect before the transition.
func localToUTC(l float64) float64 {
	if math.IsNaN(l) || l < -maxTimeBeforeUTCMs || l > maxTimeBeforeUTCMs {
		return math.NaN()
	}
	l = math.Trunc(l)
	return l - float64(icu.OffsetFromLocal(localZone(), l)*1000)
}

// ---------------------------------------------------------------------------------------------
// Date

// Date is a JavaScript Date: its time value in epoch milliseconds, NaN when invalid.
type Date struct{ t float64 }

var (
	clockMu sync.Mutex
	clockMs = math.NaN()
)

// SetNow fixes the clock that new Date() and Date.now() read, and with them dayjs() with no date:
// every later read returns ms.
func SetNow(ms float64) {
	clockMu.Lock()
	clockMs = ms
	clockMu.Unlock()
}

// now is Date.now().
func now() float64 {
	clockMu.Lock()
	defer clockMu.Unlock()
	if !math.IsNaN(clockMs) {
		return clockMs
	}
	return float64(nowMillis())
}

// NewDate is new Date(...args).
func NewDate(args ...any) *Date {
	switch len(args) {
	case 0:
		return &Date{now()}
	case 1:
		return &Date{dateFromValue(args[0])}
	}
	return &Date{timeClip(localToUTC(dateArgs(args)))}
}

// dateFromValue is the one-argument constructor.
func dateFromValue(v any) float64 {
	switch x := v.(type) {
	case *Date:
		return x.t
	case string:
		return dateParse(x)
	case *Dayjs, *Duration:
		return timeClip(toNumber(x))
	case []any, Obj, *Object:
		return dateParse(toJSString(x))
	case *big.Int:
		throw("TypeError", "Cannot convert a BigInt value to a number")
	}
	return timeClip(toNumber(v))
}

func (d *Date) clone() *Date { return &Date{d.t} }

// GetTime is Date.prototype.getTime.
func (d *Date) GetTime() float64 { return d.t }

// ValueOf is Date.prototype.valueOf.
func (d *Date) ValueOf() float64 { return d.t }

type dateFields struct {
	year, month, date, hours, minutes, seconds, ms, weekday float64
}

func posMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		r += b
	} else if r == 0 {
		return 0
	}
	return r
}

func nanFields() dateFields {
	n := math.NaN()
	return dateFields{n, n, n, n, n, n, n, n}
}

// fieldsOf splits a time value into calendar fields.
func fieldsOf(t float64) dateFields {
	if math.IsNaN(t) {
		return nanFields()
	}
	day := math.Floor(t / msPerDay)
	within := t - day*msPerDay
	y, m, dd := civilFromDays(int64(day))
	return dateFields{
		year:    float64(y),
		month:   float64(m - 1),
		date:    float64(dd),
		hours:   math.Floor(within / msPerHour),
		minutes: posMod(math.Floor(within/msPerMinute), 60),
		seconds: posMod(math.Floor(within/msPerSecond), 60),
		ms:      posMod(within, 1000),
		weekday: posMod(day+4, 7),
	}
}

func (d *Date) local() dateFields {
	if math.IsNaN(d.t) {
		return nanFields()
	}
	return fieldsOf(toLocal(d.t))
}

func (d *Date) utc() dateFields { return fieldsOf(d.t) }

func (d *Date) getFullYear() float64        { return d.local().year }
func (d *Date) getMonth() float64           { return d.local().month }
func (d *Date) getDate() float64            { return d.local().date }
func (d *Date) getDay() float64             { return d.local().weekday }
func (d *Date) getHours() float64           { return d.local().hours }
func (d *Date) getMinutes() float64         { return d.local().minutes }
func (d *Date) getSeconds() float64         { return d.local().seconds }
func (d *Date) getMilliseconds() float64    { return d.local().ms }
func (d *Date) getUTCFullYear() float64     { return d.utc().year }
func (d *Date) getUTCMonth() float64        { return d.utc().month }
func (d *Date) getUTCDate() float64         { return d.utc().date }
func (d *Date) getUTCDay() float64          { return d.utc().weekday }
func (d *Date) getUTCHours() float64        { return d.utc().hours }
func (d *Date) getUTCMinutes() float64      { return d.utc().minutes }
func (d *Date) getUTCSeconds() float64      { return d.utc().seconds }
func (d *Date) getUTCMilliseconds() float64 { return d.utc().ms }

// GetTimezoneOffset is Date.prototype.getTimezoneOffset: V8 truncates the difference between
// UTC and local time to whole minutes.
func (d *Date) GetTimezoneOffset() float64 {
	if math.IsNaN(d.t) {
		return math.NaN()
	}
	diff := int64(d.t) - int64(toLocal(d.t))
	return float64(diff / 60000)
}

// argOr reads an optional setter argument.
func argOr(args []any, i int, def float64) float64 {
	if i < len(args) {
		return toNumberStrict(args[i])
	}
	return def
}

// setField implements the Date setters. name is the setter's field (FullYear, Month, Date,
// Hours, Minutes, Seconds or Milliseconds); utc selects the setUTC variant. It stores and
// returns the new time value.
func (d *Date) setField(name string, utc bool, args ...any) float64 {
	vals := make([]float64, len(args))
	for i := range args {
		vals[i] = toNumberStrict(args[i])
	}
	arg := func(i int, def float64) float64 {
		if i < len(vals) {
			return vals[i]
		}
		return def
	}
	if len(vals) == 0 {
		vals = []float64{math.NaN()}
	}
	t := d.t
	if !utc && !math.IsNaN(t) {
		t = toLocal(t)
	}
	if math.IsNaN(t) && name != "FullYear" {
		d.t = math.NaN()
		return d.t
	}
	var f dateFields
	if math.IsNaN(t) {
		f = dateFields{month: 0, date: 1}
		t = 0
	} else {
		f = fieldsOf(t)
	}
	day := math.Floor(t / msPerDay)
	within := t - day*msPerDay
	var nt float64
	switch name {
	case "FullYear":
		nt = makeDate(makeDay(vals[0], arg(1, f.month), arg(2, f.date)), within)
	case "Month":
		nt = makeDate(makeDay(f.year, vals[0], arg(1, f.date)), within)
	case "Date":
		nt = makeDate(makeDay(f.year, f.month, vals[0]), within)
	case "Hours":
		nt = makeDate(day, makeTime(vals[0], arg(1, f.minutes), arg(2, f.seconds), arg(3, f.ms)))
	case "Minutes":
		nt = makeDate(day, makeTime(f.hours, vals[0], arg(1, f.seconds), arg(2, f.ms)))
	case "Seconds":
		nt = makeDate(day, makeTime(f.hours, f.minutes, vals[0], arg(1, f.ms)))
	case "Milliseconds":
		nt = makeDate(day, makeTime(f.hours, f.minutes, f.seconds, vals[0]))
	default:
		throw("TypeError", "Date.prototype.set"+name+" is not a function")
	}
	if utc {
		d.t = timeClip(nt)
	} else {
		d.t = timeClip(localToUTC(nt))
	}
	return d.t
}

// callSetter is date[method](...args) for a setter name such as "setUTCHours".
func (d *Date) callSetter(method string, args ...any) float64 {
	name := strings.TrimPrefix(method, "set")
	utc := strings.HasPrefix(name, "UTC")
	return d.setField(strings.TrimPrefix(name, "UTC"), utc, args...)
}

var (
	shortWeekDays = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	shortMonths   = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
)

// ToISOString is Date.prototype.toISOString.
func (d *Date) ToISOString() string {
	if math.IsNaN(d.t) {
		throw("RangeError", "Invalid time value")
	}
	f := d.utc()
	y := int64(f.year)
	var ys string
	switch {
	case y < 0:
		ys = fmt.Sprintf("-%06d", -y)
	case y > 9999:
		ys = fmt.Sprintf("+%06d", y)
	default:
		ys = fmt.Sprintf("%04d", y)
	}
	return fmt.Sprintf("%s-%02d-%02dT%02d:%02d:%02d.%03dZ", ys, int(f.month)+1, int(f.date),
		int(f.hours), int(f.minutes), int(f.seconds), int(f.ms))
}

// ToUTCString is Date.prototype.toUTCString.
func (d *Date) ToUTCString() string {
	if math.IsNaN(d.t) {
		return "Invalid Date"
	}
	f := d.utc()
	format := "%s, %02d %s %04d %02d:%02d:%02d GMT"
	if f.year < 0 {
		format = "%s, %02d %s %05d %02d:%02d:%02d GMT"
	}
	return fmt.Sprintf(format, shortWeekDays[int(f.weekday)], int(f.date), shortMonths[int(f.month)],
		int64(f.year), int(f.hours), int(f.minutes), int(f.seconds))
}

// toString is Date.prototype.toString.
func (d *Date) toString() string {
	if math.IsNaN(d.t) {
		return "Invalid Date"
	}
	f := d.local()
	off := int64(localOffset(d.t)) / 60000
	sign := '+'
	if off < 0 {
		sign = '-'
		off = -off
	}
	name, ok := icu.ZoneName(localZone(), d.t, true)
	if !ok {
		name = "Coordinated Universal Time"
	}
	format := "%s %s %02d %04d %02d:%02d:%02d GMT%c%02d%02d (%s)"
	if f.year < 0 {
		format = "%s %s %02d %05d %02d:%02d:%02d GMT%c%02d%02d (%s)"
	}
	return fmt.Sprintf(format, shortWeekDays[int(f.weekday)], shortMonths[int(f.month)], int(f.date),
		int64(f.year), int(f.hours), int(f.minutes), int(f.seconds), sign, off/60, off%60, name)
}

// civilFromDays converts days since 1970-01-01 to a proleptic Gregorian date.
func civilFromDays(z int64) (y, m, d int64) {
	z += 719468
	era := z / 146097
	if z < 0 && z%146097 != 0 {
		era--
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y = yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d = doy - (153*mp+2)/5 + 1
	if mp < 10 {
		m = mp + 3
	} else {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return
}
