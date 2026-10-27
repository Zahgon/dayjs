// Package dayjs is a Go port of Day.js 1.11.23: the core, its 37 plugins (package plugin) and its
// 143 locales (package locale), reproducing the original's results exactly, including what Day.js
// inherits from the JavaScript runtime it runs on: Date construction, parsing and local time, and
// the Intl time zone data of Node.js 24.
//
// Day.js is a dynamic library: plugins replace methods of the Dayjs prototype and wrap the ones
// they replace, so the result of a call depends on which plugins are installed and in which
// order. The port keeps that model: every method dispatches through a method table that Extend
// lets plugins rewrite, and the exported methods are typed entry points into it.
package dayjs

import (
	"math"
	"regexp"
	"strings"
)

const (
	unitMS      = "millisecond"
	unitS       = "second"
	unitMin     = "minute"
	unitH       = "hour"
	unitD       = "day"
	unitW       = "week"
	unitM       = "month"
	unitQ       = "quarter"
	unitY       = "year"
	unitDate    = "date"
	formatDflt  = "YYYY-MM-DDTHH:mm:ssZ"
	invalidDate = "Invalid Date"

	msASecond = 1e3
	msAMinute = 60 * msASecond
	msAHour   = 60 * msAMinute
	msADay    = 24 * msAHour
	msAWeek   = 7 * msADay
)

var (
	regexParse  = jsRegexp(`^(\d{4})[-/]?(\d{1,2})?[-/]?(\d{0,2})[Tt\s]*(\d{1,2})?:?(\d{1,2})?:?(\d{1,2})?[.:]?(\d+)?$`)
	regexFormat = regexp.MustCompile(`\[([^\]]+)]|YYYY|YY|M{1,4}|D{1,2}|d{1,4}|H{1,2}|h{1,2}|a|A|m{1,2}|s{1,2}|Z{1,2}|SSS`)
	regexZEnd   = regexp.MustCompile(`(?i)Z$`)
)

// Dayjs is a Day.js instance.
type Dayjs struct {
	l                          any     // $L: the locale name
	d                          *Date   // $d
	x                          *Object // $x, shared with the instances cloned from this one; nil until set
	u                          any     // $u: true, false or undefined
	offset                     any     // $offset: minutes, or undefined
	inited                     bool    // whether init has set the fields below; until then they read as undefined
	y, mo, dd, w, h, mi, s, ms float64
}

// method is a prototype method: this and the call's arguments.
type method func(this *Dayjs, args ...any) any

// config is the cfg object the constructor and the parse methods receive.
type config struct {
	date   any
	args   []any
	locale any
	utc    any
	x      any // *Object or undefined
	offset any
}

func newConfig() *config {
	return &config{date: Undefined, locale: Undefined, utc: Undefined, x: Undefined, offset: Undefined}
}

var (
	globalLocale = "en"
	loadedLocale = map[string]*Object{}
	proto        = map[string]method{}
	statics      = map[string]any{}
	installed    = map[*Plugin]bool{}
	enLocale     *Object
	customParse  bool // dayjs.p.customParseFormat
)

func init() {
	enLocale = objFrom(Obj{
		"name", "en",
		"weekdays", []any{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		"months", []any{"January", "February", "March", "April", "May", "June", "July", "August",
			"September", "October", "November", "December"},
		"ordinal", Func(func(a ...any) any {
			n := toNumber(arg(a, 0))
			s := []any{"th", "st", "nd", "rd"}
			v := math.Mod(n, 100)
			r := index(s, math.Mod(v-20, 10))
			if !truthy(r) {
				r = index(s, v)
			}
			if !truthy(r) {
				r = s[0]
			}
			return "[" + toJSString(arg(a, 0)) + toJSString(r) + "]"
		}),
	})
	loadedLocale["en"] = enLocale
	installCore()
}

// arg is arguments[i], undefined past the end.
func arg(a []any, i int) any {
	if i < len(a) {
		return a[i]
	}
	return Undefined
}

// call dispatches a prototype method.
func (d *Dayjs) call(name string, args ...any) any {
	m, ok := proto[name]
	if !ok || m == nil {
		throw("TypeError", "d."+name+" is not a function")
	}
	return m(d, args...)
}

func (d *Dayjs) callDayjs(name string, args ...any) *Dayjs {
	r := d.call(name, args...)
	if x, ok := r.(*Dayjs); ok {
		return x
	}
	throw("TypeError", "d."+name+"(...) did not return a Day.js object")
	return nil
}

// field is this[name] for the instance's own data properties.
func (d *Dayjs) field(name string) any {
	if !d.inited {
		switch name {
		case "$y", "$M", "$D", "$W", "$H", "$m", "$s", "$ms":
			return Undefined
		}
	}
	switch name {
	case "$y":
		return d.y
	case "$M":
		return d.mo
	case "$D":
		return d.dd
	case "$W":
		return d.w
	case "$H":
		return d.h
	case "$m":
		return d.mi
	case "$s":
		return d.s
	case "$ms":
		return d.ms
	case "$L":
		return d.l
	case "$u":
		return d.u
	case "$offset":
		return d.offset
	}
	return Undefined
}

// xObj is this.$x, throwing as JavaScript does when it is not set yet.
func (d *Dayjs) xObj(key string) *Object {
	if d.x == nil {
		throw("TypeError", "Cannot read properties of undefined (reading '"+key+"')")
	}
	return d.x
}

// ---------------------------------------------------------------------------------------------
// utils

// padStart is Utils.s: it returns the value itself when its string is long enough.
func padStart(v any, length float64, pad string) any {
	s := toJSString(v)
	if s == "" || float64(utf16Len(s)) >= length {
		return v
	}
	n := int(length+1) - utf16Len(s)
	return strings.Repeat(pad, n-1) + s
}

// padZoneStr is Utils.z.
func padZoneStr(d *Dayjs) string {
	negMinutes := -toNumber(d.call("utcOffset"))
	minutes := math.Abs(negMinutes)
	hourOffset := math.Floor(minutes / 60)
	minuteOffset := math.Mod(minutes, 60)
	sign := "-"
	if negMinutes <= 0 {
		sign = "+"
	}
	return sign + toJSString(padStart(hourOffset, 2, "0")) + ":" + toJSString(padStart(minuteOffset, 2, "0"))
}

// monthDiff is Utils.m, moment's month difference.
func monthDiff(a, b *Dayjs) float64 {
	if toNumber(a.call("date")) < toNumber(b.call("date")) {
		return -monthDiff(b, a)
	}
	whole := (toNumber(b.call("year"))-toNumber(a.call("year")))*12 + (toNumber(b.call("month")) - toNumber(a.call("month")))
	anchor := a.callDayjs("clone").callDayjs("add", whole, unitM)
	bv, av := toNumber(b), toNumber(anchor)
	c := bv-av < 0
	step := 1.0
	if c {
		step = -1
	}
	anchor2 := a.callDayjs("clone").callDayjs("add", whole+step, unitM)
	a2 := toNumber(anchor2)
	var den float64
	if c {
		den = av - a2
	} else {
		den = a2 - av
	}
	r := -(whole + (bv-av)/den)
	if r == 0 || math.IsNaN(r) {
		return 0
	}
	return r
}

// absFloor is Utils.a.
func absFloor(n float64) float64 {
	if n < 0 {
		c := math.Ceil(n)
		if c == 0 || math.IsNaN(c) {
			return 0
		}
		return c
	}
	return math.Floor(n)
}

var specialUnits = map[string]string{
	"M": unitM, "y": unitY, "w": unitW, "d": unitD, "D": unitDate, "h": unitH, "m": unitMin,
	"s": unitS, "ms": unitMS, "Q": unitQ,
}

// prettyUnit is Utils.p.
func prettyUnit(u any) string {
	if s, ok := specialUnits[toJSString(u)]; ok {
		return s
	}
	var str string
	if truthy(u) {
		str = toJSString(u)
	}
	str = jsToLower(str)
	return strings.TrimSuffix(str, "s")
}

// wrapper is Utils.w.
func wrapper(date any, inst *Dayjs) *Dayjs {
	cfg := newConfig()
	cfg.locale = inst.l
	cfg.utc = inst.u
	if inst.x != nil {
		cfg.x = inst.x
	}
	cfg.offset = inst.offset
	return factory(date, cfg)
}

// ---------------------------------------------------------------------------------------------
// locales

// parseLocale is dayjs.locale's implementation (Utils.l).
func parseLocale(preset, object any, isLocal bool) any {
	if !truthy(preset) {
		return globalLocale
	}
	var l any = Undefined
	if s, ok := preset.(string); ok {
		lower := jsToLower(s)
		if _, ok := loadedLocale[lower]; ok {
			l = lower
		}
		if truthy(object) {
			loadedLocale[lower] = asObject(object)
			l = lower
		}
		if isUndef(l) && len(strings.Split(s, "-")) > 1 {
			return parseLocale(strings.Split(s, "-")[0], Undefined, false)
		}
	} else {
		o := asObject(preset)
		name := toJSString(o.get("name"))
		loadedLocale[name] = o
		l = name
	}
	if !isLocal && truthy(l) {
		globalLocale = l.(string)
	}
	if truthy(l) {
		return l
	}
	if !isLocal {
		return globalLocale
	}
	return false
}

// asObject converts an object argument (Obj or *Object) to the object it denotes.
func asObject(v any) *Object {
	switch o := v.(type) {
	case *Object:
		return o
	case Obj:
		return objFrom(o)
	}
	return newObj()
}

func (d *Dayjs) localeObj() *Object {
	if s, ok := d.l.(string); ok {
		if o, ok := loadedLocale[s]; ok {
			return o
		}
	}
	return nil
}

// locProp reads a property of the instance's locale object, throwing as JavaScript does when
// the locale is not loaded.
func (d *Dayjs) locProp(key string) any {
	o := d.localeObj()
	if o == nil {
		throw("TypeError", "Cannot read properties of undefined (reading '"+key+"')")
	}
	return o.get(key)
}

// ---------------------------------------------------------------------------------------------
// construction

func isDayjs(v any) bool {
	_, ok := v.(*Dayjs)
	return ok
}

// factory is dayjs(date, c) called with both arguments, as Utils.w does.
func factory(date any, c any) *Dayjs { return construct([]any{date, c}) }

// construct is dayjs(...args).
func construct(args []any) *Dayjs {
	date := arg(args, 0)
	if d, ok := date.(*Dayjs); ok {
		return d.callDayjs("clone")
	}
	c := arg(args, 1)
	var cfg *config
	switch cv := c.(type) {
	case *config:
		cfg = cv
	case Obj, *Object:
		o := asObject(cv)
		cfg = newConfig()
		cfg.locale, cfg.utc, cfg.x, cfg.offset = o.get("locale"), o.get("utc"), o.get("x"), o.get("$offset")
	case nil:
		throw("TypeError", "Cannot set properties of null (setting 'date')")
	default:
		cfg = newConfig()
	}
	cfg.date = date
	cfg.args = args
	return newDayjs(cfg)
}

// newDayjs is new Dayjs(cfg).
func newDayjs(cfg *config) *Dayjs {
	d := newBlank()
	d.l = parseLocale(cfg.locale, Undefined, true)
	d.call("parse", cfg)
	if d.x == nil {
		if x, ok := cfg.x.(*Object); ok {
			d.x = x
		} else {
			d.x = newObj()
		}
	}
	return d
}

func newBlank() *Dayjs {
	nan := math.NaN()
	return &Dayjs{u: Undefined, offset: Undefined, y: nan, mo: nan, dd: nan, w: nan, h: nan, mi: nan, s: nan, ms: nan}
}

// parseDate is the core's parseDate.
func parseDate(cfg *config) *Date {
	date := cfg.date
	if date == nil {
		return &Date{math.NaN()}
	}
	if isUndef(date) {
		return NewDate()
	}
	if dt, ok := date.(*Date); ok {
		return &Date{dt.t}
	}
	if s, ok := date.(string); ok && !regexZEnd.MatchString(s) {
		if m := regexParse.FindStringSubmatch(s); m != nil {
			mo := 0.0
			if m[2] != "" {
				if mo = toNumber(m[2]) - 1; math.IsNaN(mo) {
					mo = 0
				}
			}
			msStr := m[7]
			if msStr == "" {
				msStr = "0"
			}
			ms := jsSubstring(msStr, 0, 3)
			or := func(s string, def any) any {
				if s == "" {
					return def
				}
				return s
			}
			parts := []any{m[1], mo, or(m[3], 1.0), or(m[4], 0.0), or(m[5], 0.0), or(m[6], 0.0), ms}
			if truthy(cfg.utc) {
				return &Date{dateUTC(parts...)}
			}
			return NewDate(parts...)
		}
	}
	return NewDate(date)
}

func installCore() {
	proto["parse"] = func(this *Dayjs, a ...any) any {
		cfg := a[0].(*config)
		this.d = parseDate(cfg)
		this.call("init")
		return Undefined
	}
	proto["init"] = func(this *Dayjs, a ...any) any {
		this.setFields(this.d.local())
		return Undefined
	}
	proto["isValid"] = func(this *Dayjs, a ...any) any { return !math.IsNaN(this.d.t) }
	proto["isSame"] = func(this *Dayjs, a ...any) any {
		other := toNumber(construct([]any{arg(a, 0)}))
		return toNumber(this.call("startOf", arg(a, 1))) <= other && other <= toNumber(this.call("endOf", arg(a, 1)))
	}
	proto["isAfter"] = func(this *Dayjs, a ...any) any {
		return toNumber(construct([]any{arg(a, 0)})) < toNumber(this.call("startOf", arg(a, 1)))
	}
	proto["isBefore"] = func(this *Dayjs, a ...any) any {
		return toNumber(this.call("endOf", arg(a, 1))) < toNumber(construct([]any{arg(a, 0)}))
	}
	proto["$g"] = func(this *Dayjs, a ...any) any {
		if isUndef(arg(a, 0)) {
			return this.field(toJSString(arg(a, 1)))
		}
		return this.call("set", arg(a, 2), arg(a, 0))
	}
	proto["unix"] = func(this *Dayjs, a ...any) any { return math.Floor(toNumber(this.call("valueOf")) / 1000) }
	proto["valueOf"] = func(this *Dayjs, a ...any) any { return this.d.GetTime() }
	proto["startOf"] = coreStartOf
	proto["endOf"] = func(this *Dayjs, a ...any) any { return this.call("startOf", arg(a, 0), false) }
	proto["$set"] = coreSet
	proto["set"] = func(this *Dayjs, a ...any) any {
		return this.callDayjs("clone").call("$set", arg(a, 0), arg(a, 1))
	}
	proto["get"] = func(this *Dayjs, a ...any) any { return this.call(prettyUnit(arg(a, 0))) }
	proto["add"] = coreAdd
	proto["subtract"] = func(this *Dayjs, a ...any) any {
		return this.call("add", toNumber(arg(a, 0))*-1, arg(a, 1))
	}
	proto["format"] = coreFormat
	proto["utcOffset"] = func(this *Dayjs, a ...any) any {
		return -mathRound(this.d.GetTimezoneOffset()/15) * 15
	}
	proto["diff"] = coreDiff
	proto["daysInMonth"] = func(this *Dayjs, a ...any) any { return this.callDayjs("endOf", unitM).dd }
	proto["$locale"] = func(this *Dayjs, a ...any) any {
		if o := this.localeObj(); o != nil {
			return o
		}
		return Undefined
	}
	proto["locale"] = func(this *Dayjs, a ...any) any {
		if !truthy(arg(a, 0)) {
			return this.l
		}
		that := this.callDayjs("clone")
		if next := parseLocale(arg(a, 0), arg(a, 1), true); truthy(next) {
			that.l = next
		}
		return that
	}
	proto["clone"] = func(this *Dayjs, a ...any) any { return wrapper(this.d, this) }
	proto["toDate"] = func(this *Dayjs, a ...any) any { return &Date{timeClip(toNumber(this.call("valueOf")))} }
	proto["toJSON"] = func(this *Dayjs, a ...any) any {
		if truthy(this.call("isValid")) {
			return this.call("toISOString")
		}
		return nil
	}
	proto["toISOString"] = func(this *Dayjs, a ...any) any { return this.d.ToISOString() }
	proto["toString"] = func(this *Dayjs, a ...any) any { return this.d.ToUTCString() }
	for _, g := range [][2]string{{"$ms", unitMS}, {"$s", unitS}, {"$m", unitMin}, {"$H", unitH},
		{"$W", unitD}, {"$M", unitM}, {"$y", unitY}, {"$D", unitDate}} {
		g := g
		proto[g[1]] = func(this *Dayjs, a ...any) any { return this.call("$g", arg(a, 0), g[0], g[1]) }
	}
	statics["unix"] = Func(func(a ...any) any { return construct([]any{toNumber(arg(a, 0)) * 1e3}) })
	statics["locale"] = Func(func(a ...any) any {
		isLocal := truthy(arg(a, 2))
		return parseLocale(arg(a, 0), arg(a, 1), isLocal)
	})
}

func coreStartOf(this *Dayjs, a ...any) any {
	isStartOf := true
	if so := arg(a, 1); !isUndef(so) {
		isStartOf = truthy(so)
	}
	unit := prettyUnit(arg(a, 0))
	instanceFactory := func(d, m float64) *Dayjs {
		var ins *Dayjs
		if truthy(this.u) {
			ins = wrapper(dateUTC(this.y, m, d), this)
		} else {
			ins = wrapper(NewDate(this.y, m, d), this)
		}
		if isStartOf {
			return ins
		}
		return ins.callDayjs("endOf", unitD)
	}
	instanceFactorySet := func(method string, slice int) *Dayjs {
		start := []any{0.0, 0.0, 0.0, 0.0}
		end := []any{23.0, 59.0, 59.0, 999.0}
		vals := end
		if isStartOf {
			vals = start
		}
		target := this.call("toDate", "s").(*Date)
		return wrapper(target.callSetter(method, vals[slice:]...), this)
	}
	utcPad := "set"
	if truthy(this.u) {
		utcPad = "setUTC"
	}
	switch unit {
	case unitY:
		if isStartOf {
			return instanceFactory(1, 0)
		}
		return instanceFactory(31, 11)
	case unitM:
		if isStartOf {
			return instanceFactory(1, this.mo)
		}
		return instanceFactory(0, this.mo+1)
	case unitW:
		weekStart := toNumber(orZero(this.locProp("weekStart")))
		gap := this.w
		if this.w < weekStart {
			gap += 7
		}
		gap -= weekStart
		if isStartOf {
			return instanceFactory(this.dd-gap, this.mo)
		}
		return instanceFactory(this.dd+(6-gap), this.mo)
	case unitD, unitDate:
		return instanceFactorySet(utcPad+"Hours", 0)
	case unitH:
		return instanceFactorySet(utcPad+"Minutes", 1)
	case unitMin:
		return instanceFactorySet(utcPad+"Seconds", 2)
	case unitS:
		return instanceFactorySet(utcPad+"Milliseconds", 3)
	}
	return this.call("clone")
}

func (d *Dayjs) setFields(f dateFields) {
	d.y, d.mo, d.dd, d.w = f.year, f.month, f.date, f.weekday
	d.h, d.mi, d.s, d.ms = f.hours, f.minutes, f.seconds, f.ms
	d.inited = true
}

// orZero is `v || 0`.
func orZero(v any) any {
	if truthy(v) {
		return v
	}
	return 0.0
}

func coreSet(this *Dayjs, a ...any) any {
	unit := prettyUnit(arg(a, 0))
	val := arg(a, 1)
	pad := ""
	if truthy(this.u) {
		pad = "UTC"
	}
	names := map[string]string{
		unitD: "Date", unitDate: "Date", unitM: "Month", unitY: "FullYear", unitH: "Hours",
		unitMin: "Minutes", unitS: "Seconds", unitMS: "Milliseconds",
	}
	name, ok := names[unit]
	var argv any = val
	if unit == unitD {
		argv = this.dd + (toNumber(val) - this.w)
	}
	if unit == unitM || unit == unitY {
		date := this.callDayjs("clone").callDayjs("set", unitDate, 1.0)
		date.d.callSetter("set"+pad+name, argv)
		date.call("init")
		dim := toNumber(date.call("daysInMonth"))
		this.d = date.callDayjs("set", unitDate, math.Min(this.dd, dim)).d
	} else if ok {
		this.d.callSetter("set"+pad+name, argv)
	}
	this.call("init")
	return this
}

func coreAdd(this *Dayjs, a ...any) any {
	number := toNumber(arg(a, 0))
	unit := prettyUnit(arg(a, 1))
	instanceFactorySet := func(n float64) *Dayjs {
		d := construct([]any{this})
		return wrapper(d.callDayjs("date", toNumber(d.call("date"))+mathRound(n*number)), this)
	}
	switch unit {
	case unitM:
		return this.call("set", unitM, this.mo+number)
	case unitY:
		return this.call("set", unitY, this.y+number)
	case unitD:
		return instanceFactorySet(1)
	case unitW:
		return instanceFactorySet(7)
	}
	step := 1.0
	switch unit {
	case unitMin:
		step = msAMinute
	case unitH:
		step = msAHour
	case unitS:
		step = msASecond
	}
	return wrapper(this.d.GetTime()+number*step, this)
}

func coreDiff(this *Dayjs, a ...any) any {
	unit := prettyUnit(arg(a, 1))
	that := construct([]any{arg(a, 0)})
	zoneDelta := (toNumber(that.call("utcOffset")) - toNumber(this.call("utcOffset"))) * msAMinute
	diff := toNumber(this) - toNumber(that)
	var result float64
	switch unit {
	case unitY:
		result = monthDiff(this, that) / 12
	case unitM:
		result = monthDiff(this, that)
	case unitQ:
		result = monthDiff(this, that) / 3
	case unitW:
		result = (diff - zoneDelta) / msAWeek
	case unitD:
		result = (diff - zoneDelta) / msADay
	case unitH:
		result = diff / msAHour
	case unitMin:
		result = diff / msAMinute
	case unitS:
		result = diff / msASecond
	default:
		result = diff
	}
	if truthy(arg(a, 2)) {
		return result
	}
	return absFloor(result)
}

func coreFormat(this *Dayjs, a ...any) any {
	locale := this.localeObj()
	if locale == nil {
		throw("TypeError", "Cannot read properties of undefined (reading 'invalidDate')")
	}
	if !truthy(this.call("isValid")) {
		if inv := locale.get("invalidDate"); truthy(inv) {
			return inv
		}
		return invalidDate
	}
	str := formatDflt
	if f := arg(a, 0); truthy(f) {
		str = toJSString(f)
	}
	zoneStr := padZoneStr(this)
	weekdays, months, meridiem := locale.get("weekdays"), locale.get("months"), locale.get("meridiem")
	getShort := func(arr any, idx float64, full any, length any) any {
		if truthy(arr) {
			v := index(arr, idx)
			if !truthy(v) {
				v = call(arr, "arr", this, str)
			}
			if truthy(v) {
				return v
			}
		}
		return jsSliceAny(index(full, idx), 0, length)
	}
	getH := func(num float64) any {
		h := math.Mod(this.h, 12)
		if h == 0 || math.IsNaN(h) {
			h = 12
		}
		return padStart(h, num, "0")
	}
	meridiemFunc := func(hour, minute float64, lower bool) any {
		if truthy(meridiem) {
			return call(meridiem, "meridiem", hour, minute, lower)
		}
		m := "PM"
		if hour < 12 {
			m = "AM"
		}
		if lower {
			return strings.ToLower(m)
		}
		return m
	}
	matches := func(match string) any {
		switch match {
		case "YY":
			return jsSliceAny(toJSString(this.y), -2, Undefined)
		case "YYYY":
			return padStart(this.y, 4, "0")
		case "M":
			return this.mo + 1
		case "MM":
			return padStart(this.mo+1, 2, "0")
		case "MMM":
			return getShort(locale.get("monthsShort"), this.mo, months, 3.0)
		case "MMMM":
			return getShort(months, this.mo, Undefined, Undefined)
		case "D":
			return this.dd
		case "DD":
			return padStart(this.dd, 2, "0")
		case "d":
			return toJSString(this.w)
		case "dd":
			return getShort(locale.get("weekdaysMin"), this.w, weekdays, 2.0)
		case "ddd":
			return getShort(locale.get("weekdaysShort"), this.w, weekdays, 3.0)
		case "dddd":
			return index(weekdays, this.w)
		case "H":
			return toJSString(this.h)
		case "HH":
			return padStart(this.h, 2, "0")
		case "h":
			return getH(1)
		case "hh":
			return getH(2)
		case "a":
			return meridiemFunc(this.h, this.mi, true)
		case "A":
			return meridiemFunc(this.h, this.mi, false)
		case "m":
			return toJSString(this.mi)
		case "mm":
			return padStart(this.mi, 2, "0")
		case "s":
			return toJSString(this.s)
		case "ss":
			return padStart(this.s, 2, "0")
		case "SSS":
			return padStart(this.ms, 3, "0")
		case "Z":
			return zoneStr
		}
		return nil
	}
	return replaceAllSubmatchFunc(regexFormat, str, func(m []string, has []bool) string {
		if has[1] && m[1] != "" {
			return m[1]
		}
		if r := matches(m[0]); truthy(r) {
			return toJSString(r)
		}
		return strings.Replace(zoneStr, ":", "", 1)
	})
}
