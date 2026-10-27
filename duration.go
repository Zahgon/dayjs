package dayjs

import (
	"math"
	"regexp"
)

const (
	msAYear  = msADay * 365
	msAMonth = msAYear / 12
)

var (
	durationParse  = regexp.MustCompile(`^(-|\+)?P(?:([-+]?[0-9,.]*)Y)?(?:([-+]?[0-9,.]*)M)?(?:([-+]?[0-9,.]*)W)?(?:([-+]?[0-9,.]*)D)?(?:T(?:([-+]?[0-9,.]*)H)?(?:([-+]?[0-9,.]*)M)?(?:([-+]?[0-9,.]*)S)?)?$`)
	durationFormat = regexp.MustCompile(`\[([^\]]+)]|YYYY|YY|Y|M{1,2}|D{1,2}|H{1,2}|m{1,2}|s{1,2}|SSS`)

	unitToMS = map[string]float64{
		"years": msAYear, "months": msAMonth, "days": msADay, "hours": msAHour, "minutes": msAMinute,
		"seconds": msASecond, "milliseconds": 1, "weeks": msAWeek,
	}
)

// Duration is a duration created by the duration plugin's dayjs.duration.
type Duration struct {
	d  *Object // $d
	ms any     // $ms: a number, or undefined when the input was not understood
	l  any     // $l
}

func durPretty(unit any) string { return prettyUnit(unit) + "s" }

func durUnitMS(unit string) float64 {
	if v, ok := unitToMS[unit]; ok {
		return v
	}
	return math.NaN()
}

func roundNumber(n float64) float64 {
	if n < 0 {
		return math.Ceil(n)
	}
	return math.Floor(n)
}

// newDuration is new Duration(input, unit, locale).
func newDuration(input, unit, locale any) *Duration {
	t := &Duration{d: newObj(), ms: Undefined, l: locale}
	if isUndef(input) {
		t.ms = 0.0
		t.parseFromMilliseconds()
	}
	if truthy(unit) {
		return newDuration(toNumber(input)*durUnitMS(durPretty(unit)), Undefined, t.l)
	}
	switch v := input.(type) {
	case float64:
		t.ms = v
		t.parseFromMilliseconds()
		return t
	case nil:
		throw("TypeError", "Cannot convert undefined or null to object")
	case string:
		if m := durationParse.FindStringSubmatch(v); m != nil {
			idx := durationParse.FindStringSubmatchIndex(v)
			nums := make([]float64, 7)
			for i := 0; i < 7; i++ {
				if idx[2*(i+2)] >= 0 {
					nums[i] = stringToNumber(m[i+2])
				}
			}
			for i, k := range []string{"years", "months", "weeks", "days", "hours", "minutes", "seconds"} {
				t.d.set(k, nums[i])
			}
			t.calMilliseconds()
		}
		return t
	case undefinedType, bool, Func, *Function:
		return t
	default:
		if typeOf(v) == "object" {
			for _, k := range objectKeys(v) {
				t.d.set(durPretty(k), propOf(v, k))
			}
			t.calMilliseconds()
		}
	}
	return t
}

func (t *Duration) calMilliseconds() {
	total := 0.0
	for _, k := range t.d.keys {
		total += toNumber(orZero(t.d.get(k))) * durUnitMS(k)
	}
	t.ms = total
}

func (t *Duration) parseFromMilliseconds() {
	ms := toNumber(t.ms)
	t.d.set("years", roundNumber(ms/msAYear))
	ms = math.Mod(ms, msAYear)
	t.d.set("months", roundNumber(ms/msAMonth))
	ms = math.Mod(ms, msAMonth)
	t.d.set("days", roundNumber(ms/msADay))
	ms = math.Mod(ms, msADay)
	t.d.set("hours", roundNumber(ms/msAHour))
	ms = math.Mod(ms, msAHour)
	t.d.set("minutes", roundNumber(ms/msAMinute))
	ms = math.Mod(ms, msAMinute)
	t.d.set("seconds", roundNumber(ms/msASecond))
	ms = math.Mod(ms, msASecond)
	t.d.set("milliseconds", ms)
}

type numberUnit struct {
	negative bool
	format   string
}

func getNumberUnitFormat(number any, unit string) numberUnit {
	if !truthy(number) {
		return numberUnit{}
	}
	if toNumber(number) < 0 {
		return numberUnit{true, numberToString(math.Abs(toNumber(number))) + unit}
	}
	return numberUnit{false, toJSString(number) + unit}
}

func durWrapper(input any, t *Duration) *Duration { return newDuration(input, Undefined, t.l) }

// ToISOString is duration.toISOString().
func (t *Duration) ToISOString() string {
	Y := getNumberUnitFormat(t.d.get("years"), "Y")
	M := getNumberUnitFormat(t.d.get("months"), "M")
	var days any = orZero(toNumber(t.d.get("days")))
	if w := t.d.get("weeks"); truthy(w) {
		days = jsAdd(days, toNumber(w)*7)
	}
	D := getNumberUnitFormat(days, "D")
	H := getNumberUnitFormat(t.d.get("hours"), "H")
	m := getNumberUnitFormat(t.d.get("minutes"), "M")
	seconds := orZero(t.d.get("seconds"))
	if ms := t.d.get("milliseconds"); truthy(ms) {
		seconds = jsAdd(seconds, toNumber(ms)/1000)
		seconds = mathRound(toNumber(seconds)*1000) / 1000
	}
	S := getNumberUnitFormat(seconds, "S")
	negative := Y.negative || M.negative || D.negative || H.negative || m.negative || S.negative
	T := ""
	if H.format != "" || m.format != "" || S.format != "" {
		T = "T"
	}
	P := ""
	if negative {
		P = "-"
	}
	result := P + "P" + Y.format + M.format + D.format + T + H.format + m.format + S.format
	if result == "P" || result == "-P" {
		return "P0D"
	}
	return result
}

// ToJSON is duration.toJSON().
func (t *Duration) ToJSON() string { return t.ToISOString() }

// Format is duration.format(formatStr).
func (t *Duration) Format(formatStr ...any) string {
	str := "YYYY-MM-DDTHH:mm:ss"
	if f := arg(formatStr, 0); truthy(f) {
		str = toJSString(f)
	}
	get := t.d.get
	matches := map[string]any{
		"Y": get("years"), "YY": padStart(get("years"), 2, "0"), "YYYY": padStart(get("years"), 4, "0"),
		"M": get("months"), "MM": padStart(get("months"), 2, "0"),
		"D": get("days"), "DD": padStart(get("days"), 2, "0"),
		"H": get("hours"), "HH": padStart(get("hours"), 2, "0"),
		"m": get("minutes"), "mm": padStart(get("minutes"), 2, "0"),
		"s": get("seconds"), "ss": padStart(get("seconds"), 2, "0"),
		"SSS": padStart(get("milliseconds"), 3, "0"),
	}
	return replaceAllSubmatchFunc(durationFormat, str, func(m []string, has []bool) string {
		if has[1] && m[1] != "" {
			return m[1]
		}
		return toJSString(matches[m[0]])
	})
}

// As is duration.as(unit).
func (t *Duration) As(unit any) float64 { return toNumber(t.ms) / durUnitMS(durPretty(unit)) }

// Get is duration.get(unit).
func (t *Duration) Get(unit any) float64 {
	var base any = t.ms
	switch pu := durPretty(unit); pu {
	case "milliseconds":
		base = math.Mod(toNumber(base), 1000)
	case "weeks":
		base = roundNumber(toNumber(base) / unitToMS[pu])
	default:
		base = t.d.get(pu)
	}
	if !truthy(base) {
		return 0
	}
	return toNumber(base)
}

func (t *Duration) add(input, unit, isSubtract any) *Duration {
	var another any
	switch {
	case truthy(unit):
		another = toNumber(input) * durUnitMS(durPretty(unit))
	default:
		if d, ok := input.(*Duration); ok {
			another = d.ms
		} else {
			another = durWrapper(input, t).ms
		}
	}
	sign := 1.0
	if truthy(isSubtract) {
		sign = -1
	}
	return durWrapper(jsAdd(t.ms, toNumber(another)*sign), t)
}

// Add is duration.add(input, unit, isSubtract).
func (t *Duration) Add(args ...any) *Duration { return t.add(arg(args, 0), arg(args, 1), arg(args, 2)) }

// Subtract is duration.subtract(input, unit).
func (t *Duration) Subtract(args ...any) *Duration { return t.add(arg(args, 0), arg(args, 1), true) }

// Locale is duration.locale(l).
func (t *Duration) Locale(l any) *Duration {
	that := t.Clone()
	that.l = l
	return that
}

// Clone is duration.clone().
func (t *Duration) Clone() *Duration { return durWrapper(t.ms, t) }

// Humanize is duration.humanize(withSuffix); it needs the relativeTime plugin.
func (t *Duration) Humanize(withSuffix ...any) string {
	d := construct(nil).callDayjs("add", t.ms, "ms").call("locale", t.l)
	dj, ok := d.(*Dayjs)
	if !ok {
		throw("TypeError", "(intermediate value).add(...).locale(...).fromNow is not a function")
	}
	return toJSString(dj.call("fromNow", !truthy(arg(withSuffix, 0))))
}

// ValueOf is duration.valueOf().
func (t *Duration) ValueOf() float64 { return t.AsMilliseconds() }

// Milliseconds is duration.milliseconds().
func (t *Duration) Milliseconds() float64 { return t.Get("milliseconds") }

// AsMilliseconds is duration.asMilliseconds().
func (t *Duration) AsMilliseconds() float64 { return t.As("milliseconds") }

// Seconds is duration.seconds().
func (t *Duration) Seconds() float64 { return t.Get("seconds") }

// AsSeconds is duration.asSeconds().
func (t *Duration) AsSeconds() float64 { return t.As("seconds") }

// Minutes is duration.minutes().
func (t *Duration) Minutes() float64 { return t.Get("minutes") }

// AsMinutes is duration.asMinutes().
func (t *Duration) AsMinutes() float64 { return t.As("minutes") }

// Hours is duration.hours().
func (t *Duration) Hours() float64 { return t.Get("hours") }

// AsHours is duration.asHours().
func (t *Duration) AsHours() float64 { return t.As("hours") }

// Days is duration.days().
func (t *Duration) Days() float64 { return t.Get("days") }

// AsDays is duration.asDays().
func (t *Duration) AsDays() float64 { return t.As("days") }

// Weeks is duration.weeks().
func (t *Duration) Weeks() float64 { return t.Get("weeks") }

// AsWeeks is duration.asWeeks().
func (t *Duration) AsWeeks() float64 { return t.As("weeks") }

// Months is duration.months().
func (t *Duration) Months() float64 { return t.Get("months") }

// AsMonths is duration.asMonths().
func (t *Duration) AsMonths() float64 { return t.As("months") }

// Years is duration.years().
func (t *Duration) Years() float64 { return t.Get("years") }

// AsYears is duration.asYears().
func (t *Duration) AsYears() float64 { return t.As("years") }

func init() {
	definePlugin("duration", func(o any) {
		statics["duration"] = Func(func(a ...any) any {
			l := staticCall("locale")
			return newDuration(arg(a, 0), arg(a, 1), l)
		})
		statics["isDuration"] = Func(func(a ...any) any {
			_, ok := arg(a, 0).(*Duration)
			return ok
		})
		manipulate := func(date *Dayjs, d *Duration, k float64) any {
			return date.callDayjs("add", d.Years()*k, "y").callDayjs("add", d.Months()*k, "M").
				callDayjs("add", d.Days()*k, "d").callDayjs("add", d.Hours()*k, "h").
				callDayjs("add", d.Minutes()*k, "m").callDayjs("add", d.Seconds()*k, "s").
				call("add", d.Milliseconds()*k, "ms")
		}
		oldAdd, oldSubtract := proto["add"], proto["subtract"]
		proto["add"] = func(this *Dayjs, a ...any) any {
			if d, ok := arg(a, 0).(*Duration); ok {
				return manipulate(this, d, 1)
			}
			return oldAdd(this, arg(a, 0), arg(a, 1))
		}
		proto["subtract"] = func(this *Dayjs, a ...any) any {
			if d, ok := arg(a, 0).(*Duration); ok {
				return manipulate(this, d, -1)
			}
			return oldSubtract(this, arg(a, 0), arg(a, 1))
		}
	})
}
