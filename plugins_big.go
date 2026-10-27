package dayjs

import (
	"math"
	"regexp"
	"strings"
)

var (
	reValidOffset = regexp.MustCompile(`[+-]\d\d(?::?\d\d)?`)
	reOffsetHM    = regexp.MustCompile(`([+-]|\d\d)`)
	reLocalized   = regexp.MustCompile(`(\[[^\]]+])|(LTS?|l{1,4}|L{1,4})`)
	reLocalizedT  = regexp.MustCompile(`(\[[^\]]+])|(MMMM|MM|DD|dddd)`)

	englishFormats = objFrom(Obj{
		"LTS", "h:mm:ss A",
		"LT", "h:mm A",
		"L", "MM/DD/YYYY",
		"LL", "MMMM D, YYYY",
		"LLL", "MMMM D, YYYY h:mm A",
		"LLLL", "dddd, MMMM D, YYYY h:mm A",
	})

	// fromToBase is the relativeTime plugin's proto.fromToBase, which preParsePostFormat wraps.
	fromToBase func(input, withoutSuffix any, instance *Dayjs, isFrom bool, postFormat any) any
)

// utcOffsetFromString is the utc plugin's offsetFromString: nil when the string holds no offset.
func utcOffsetFromString(v string) any {
	m := reValidOffset.FindStringIndex(v)
	if m == nil {
		return nil
	}
	parts := reOffsetHM.FindAllString(v[m[0]:m[1]], -1)
	part := func(i int) any {
		if i < len(parts) {
			return parts[i]
		}
		return Undefined
	}
	total := toNumber(part(1))*60 + toNumber(part(2))
	if total == 0 {
		return 0.0
	}
	if strictEq(part(0), "+") {
		return total
	}
	return -total
}

// localizedT is the localizedFormat utils' t.
func localizedT(format any) string {
	s, ok := format.(string)
	if !ok {
		propOf(format, "replace")
		throw("TypeError", "format.replace is not a function")
	}
	return replaceAllSubmatchFunc(reLocalizedT, s, func(m []string, has []bool) string {
		if has[1] && m[1] != "" {
			return m[1]
		}
		return jsSlice(m[2], 1.0, Undefined)
	})
}

// localizedU is the localizedFormat utils' u.
func localizedU(formatStr any, formats any) string {
	s, ok := formatStr.(string)
	if !ok {
		propOf(formatStr, "replace")
		throw("TypeError", "formatStr.replace is not a function")
	}
	return replaceAllSubmatchFunc(reLocalized, s, func(m []string, has []bool) string {
		if has[1] && m[1] != "" {
			return m[1]
		}
		b := m[2]
		if v := propOf(formats, b); truthy(v) {
			return toJSString(v)
		}
		if v := englishFormats.get(b); truthy(v) {
			return toJSString(v)
		}
		return localizedT(propOf(formats, strings.ToUpper(b)))
	})
}

func uOf(v any) any {
	if d, ok := v.(*Dayjs); ok {
		return d.u
	}
	return Undefined
}

var typeToPos = map[string]int{"year": 0, "month": 1, "day": 2, "hour": 3, "minute": 4, "second": 5}

func init() {
	definePlugin("utc", func(o any) {
		statics["utc"] = Func(func(a ...any) any {
			cfg := newConfig()
			cfg.date, cfg.utc, cfg.args = arg(a, 0), true, a
			return newDayjs(cfg)
		})
		proto["utc"] = func(this *Dayjs, a ...any) any {
			cfg := newConfig()
			cfg.locale, cfg.utc = this.l, true
			ins := factory(this.call("toDate"), cfg)
			if truthy(arg(a, 0)) {
				return ins.call("add", this.call("utcOffset"), unitMin)
			}
			return ins
		}
		proto["local"] = func(this *Dayjs, a ...any) any {
			cfg := newConfig()
			cfg.locale, cfg.utc = this.l, false
			return factory(this.call("toDate"), cfg)
		}
		oldParse := proto["parse"]
		proto["parse"] = func(this *Dayjs, a ...any) any {
			cfg := a[0].(*config)
			if truthy(cfg.utc) {
				this.u = true
			}
			if !isUndef(cfg.offset) {
				this.offset = cfg.offset
			}
			oldParse(this, cfg)
			return Undefined
		}
		oldInit := proto["init"]
		proto["init"] = func(this *Dayjs, a ...any) any {
			if truthy(this.u) {
				this.setFields(this.d.utc())
			} else {
				oldInit(this)
			}
			return Undefined
		}
		oldUtcOffset := proto["utcOffset"]
		proto["utcOffset"] = func(this *Dayjs, a ...any) any {
			input, keepLocalTime := arg(a, 0), arg(a, 1)
			if isUndef(input) {
				if truthy(this.u) {
					return 0.0
				}
				if !isUndef(this.offset) {
					return this.offset
				}
				return oldUtcOffset(this)
			}
			if s, ok := input.(string); ok {
				input = utcOffsetFromString(s)
				if input == nil {
					return this
				}
			}
			n := toNumber(input)
			offset := n
			if math.Abs(n) <= 16 {
				offset = n * 60
			}
			if offset == 0 {
				return this.call("utc", keepLocalTime)
			}
			ins := this.callDayjs("clone")
			if truthy(keepLocalTime) {
				ins.offset = offset
				ins.u = false
				return ins
			}
			var localTimezoneOffset float64
			if truthy(this.u) {
				localTimezoneOffset = this.call("toDate").(*Date).GetTimezoneOffset()
			} else {
				localTimezoneOffset = -1 * toNumber(this.call("utcOffset"))
			}
			ins = this.callDayjs("local").callDayjs("add", offset+localTimezoneOffset, unitMin)
			ins.offset = offset
			ins.xObj("$localOffset").set("$localOffset", localTimezoneOffset)
			return ins
		}
		oldFormat := proto["format"]
		proto["format"] = func(this *Dayjs, a ...any) any {
			str := arg(a, 0)
			if !truthy(str) {
				if truthy(this.u) {
					str = "YYYY-MM-DDTHH:mm:ss[Z]"
				} else {
					str = ""
				}
			}
			return oldFormat(this, str)
		}
		proto["valueOf"] = func(this *Dayjs, a ...any) any {
			added := 0.0
			if !isUndef(this.offset) {
				var local any = this.xObj("$localOffset").get("$localOffset")
				if !truthy(local) {
					local = this.d.GetTimezoneOffset()
				}
				added = toNumber(jsAdd(this.offset, local))
			}
			return this.d.ValueOf() - added*msAMinute
		}
		proto["isUTC"] = func(this *Dayjs, a ...any) any { return truthy(this.u) }
		proto["toISOString"] = func(this *Dayjs, a ...any) any { return this.call("toDate").(*Date).ToISOString() }
		proto["toString"] = func(this *Dayjs, a ...any) any { return this.call("toDate").(*Date).ToUTCString() }
		oldToDate := proto["toDate"]
		proto["toDate"] = func(this *Dayjs, a ...any) any {
			if strictEq(arg(a, 0), "s") && truthy(this.offset) {
				return construct1(this.call("format", "YYYY-MM-DD HH:mm:ss:SSS")).call("toDate")
			}
			return oldToDate(this)
		}
		oldDiff := proto["diff"]
		proto["diff"] = func(this *Dayjs, a ...any) any {
			input := arg(a, 0)
			if truthy(input) && strictEq(this.u, uOf(input)) {
				return oldDiff(this, input, arg(a, 1), arg(a, 2))
			}
			localThis := this.callDayjs("local")
			localInput := construct1(input).callDayjs("local")
			return oldDiff(localThis, localInput, arg(a, 1), arg(a, 2))
		}
	})

	definePlugin("timezone", func(o any) {
		var defaultTimezone any = Undefined
		makeFormatParts := func(timestamp float64, timezone any, timeZoneName any) []dtfPart {
			name := "short"
			if truthy(timeZoneName) {
				name = toJSString(timeZoneName)
			}
			return formatToPartsEnUS(timeClip(timestamp), timezone, name)
		}
		tzOffset := func(timestamp float64, timezone any) float64 {
			parts := makeFormatParts(timestamp, timezone, Undefined)
			filled := [6]any{Undefined, Undefined, Undefined, Undefined, Undefined, Undefined}
			for _, p := range parts {
				if pos, ok := typeToPos[p.Type]; ok {
					filled[pos] = parseInt10(p.Value)
				}
			}
			hour := filled[3]
			if strictEq(hour, 24.0) {
				hour = 0.0
			}
			utcString := toJSString(filled[0]) + "-" + toJSString(filled[1]) + "-" + toJSString(filled[2]) + " " +
				toJSString(hour) + ":" + toJSString(filled[4]) + ":" + toJSString(filled[5]) + ":000"
			utcTs := toNumber(staticDayjs("utc", utcString).call("valueOf"))
			asTS := timestamp
			over := math.Mod(asTS, 1000)
			asTS -= over
			return (utcTs - asTS) / (60 * 1000)
		}
		fixOffset := func(localTS, o0 float64, tz any) (float64, float64) {
			utcGuess := localTS - o0*60*1000
			o2 := tzOffset(utcGuess, tz)
			if o0 == o2 {
				return utcGuess, o0
			}
			utcGuess -= (o2 - o0) * 60 * 1000
			o3 := tzOffset(utcGuess, tz)
			if o2 == o3 {
				return utcGuess, o2
			}
			return localTS - math.Min(o2, o3)*60*1000, math.Max(o2, o3)
		}
		proto["tz"] = func(this *Dayjs, a ...any) any {
			timezone, keepLocalTime := arg(a, 0), arg(a, 1)
			if isUndef(timezone) {
				timezone = defaultTimezone
			}
			if !truthy(this.call("isValid")) {
				return this
			}
			oldOffset := toNumber(this.call("utcOffset"))
			date := this.call("toDate").(*Date)
			target := toLocaleStringEnUS(date.t, timezone)
			offset := tzOffset(date.t, timezone)
			var ins *Dayjs
			if offset == 0 || math.IsNaN(offset) {
				ins = this.callDayjs("utcOffset", 0.0, keepLocalTime)
			} else {
				cfg := newConfig()
				cfg.locale = this.l
				ins = factory(target, cfg).callDayjs("$set", unitMS, this.ms).callDayjs("utcOffset", offset, true)
				if truthy(keepLocalTime) {
					newOffset := toNumber(ins.call("utcOffset"))
					ins = ins.callDayjs("add", oldOffset-newOffset, unitMin)
				}
			}
			ins.xObj("$timezone").set("$timezone", timezone)
			return ins
		}
		proto["offsetName"] = func(this *Dayjs, a ...any) any {
			zone := this.xObj("$timezone").get("$timezone")
			if !truthy(zone) {
				zone = call(propOf(statics["tz"], "guess"), "d.tz.guess")
			}
			parts := makeFormatParts(toNumber(this.call("valueOf")), zone, arg(a, 0))
			for _, p := range parts {
				if jsToLower(p.Type) == "timezonename" {
					return p.Value
				}
			}
			return Undefined
		}
		oldStartOf := proto["startOf"]
		proto["startOf"] = func(this *Dayjs, a ...any) any {
			if this.x == nil || !truthy(this.x.get("$timezone")) {
				return oldStartOf(this, arg(a, 0), arg(a, 1))
			}
			cfg := newConfig()
			cfg.locale = this.l
			withoutTz := factory(this.call("format", "YYYY-MM-DD HH:mm:ss:SSS"), cfg)
			startOfWithoutTz := oldStartOf(withoutTz, arg(a, 0), arg(a, 1)).(*Dayjs)
			return startOfWithoutTz.call("tz", this.x.get("$timezone"), true)
		}
		tz := NewFunction(func(a ...any) any {
			input, arg1, arg2 := arg(a, 0), arg(a, 1), arg(a, 2)
			parseFormat := arg2
			if truthy(arg2) {
				parseFormat = arg1
			}
			timezone := arg2
			if !truthy(timezone) {
				timezone = arg1
			}
			if !truthy(timezone) {
				timezone = defaultTimezone
			}
			previousOffset := tzOffset(toNumber(construct(nil)), timezone)
			if _, ok := input.(string); !ok {
				return construct1(input).call("tz", timezone)
			}
			localTs := toNumber(staticDayjs("utc", input, parseFormat).call("valueOf"))
			if math.IsNaN(localTs) {
				ins := construct1(math.NaN())
				ins.x.set("$timezone", timezone)
				return ins
			}
			targetTs, targetOffset := fixOffset(localTs, previousOffset, timezone)
			ins := construct1(targetTs).callDayjs("utcOffset", targetOffset)
			ins.x.set("$timezone", timezone)
			return ins
		}, nil)
		tz.Props.set("guess", Func(func(a ...any) any { return resolvedTimeZone() }))
		tz.Props.set("setDefault", Func(func(a ...any) any {
			defaultTimezone = arg(a, 0)
			return Undefined
		}))
		statics["tz"] = tz
	})

	definePlugin("localizedFormat", func(o any) {
		oldFormat := proto["format"]
		enLocale.set("formats", englishFormats)
		proto["format"] = func(this *Dayjs, a ...any) any {
			formatStr := arg(a, 0)
			if isUndef(formatStr) {
				formatStr = formatDflt
			}
			loc := this.call("$locale")
			if isUndef(loc) {
				throw("TypeError", "Cannot destructure property 'formats' of 'this.$locale(...)' as it is undefined.")
			}
			formats := propOf(loc, "formats")
			if isUndef(formats) {
				formats = newObj()
			}
			return oldFormat(this, localizedU(formatStr, formats))
		}
	})

	definePlugin("relativeTime", func(o any) {
		opts := o
		if !truthy(opts) {
			opts = newObj()
		}
		relObj := objFrom(Obj{
			"future", "in %s", "past", "%s ago", "s", "a few seconds", "m", "a minute", "mm", "%d minutes",
			"h", "an hour", "hh", "%d hours", "d", "a day", "dd", "%d days", "M", "a month", "MM", "%d months",
			"y", "a year", "yy", "%d years",
		})
		enLocale.set("relativeTime", relObj)
		th := func(l string, r any, d any) *Object { return objFrom(Obj{"l", l, "r", r, "d", d}) }
		fromToBase = func(input, withoutSuffix any, instance *Dayjs, isFrom bool, postFormat any) any {
			loc := instance.locProp("relativeTime")
			if !truthy(loc) {
				loc = relObj
			}
			T := propOf(opts, "thresholds")
			if !truthy(T) {
				T = []any{
					th("s", 44.0, unitS), th("m", 89.0, Undefined), th("mm", 44.0, unitMin), th("h", 89.0, Undefined),
					th("hh", 21.0, unitH), th("d", 35.0, Undefined), th("dd", 25.0, unitD), th("M", 45.0, Undefined),
					th("MM", 10.0, unitM), th("y", 17.0, Undefined), th("yy", Undefined, unitY),
				}
			}
			tl := int(toNumber(propOf(T, "length")))
			var result any = Undefined
			var out any = Undefined
			isFuture := false
			for i := 0; i < tl; i++ {
				t := index(T, float64(i))
				if d := propOf(t, "d"); truthy(d) {
					if isFrom {
						result = construct1(input).call("diff", instance, d, true)
					} else {
						result = instance.call("diff", input, d, true)
					}
				}
				var abs any
				absIn := math.Abs(toNumber(result))
				if rounding := propOf(opts, "rounding"); truthy(rounding) {
					abs = call(rounding, "o.rounding", absIn)
				} else {
					abs = mathRound(absIn)
				}
				isFuture = toNumber(result) > 0
				r := propOf(t, "r")
				if toNumber(abs) <= toNumber(r) || !truthy(r) {
					if toNumber(abs) <= 1 && i > 0 {
						t = index(T, float64(i-1))
					}
					format := propOf(loc, toJSString(propOf(t, "l")))
					if truthy(postFormat) {
						abs = call(postFormat, "postFormat", toJSString(abs))
					}
					if fs, ok := format.(string); ok {
						out = replaceFirst(fs, "%d", abs)
					} else {
						out = call(format, "format", abs, withoutSuffix, propOf(t, "l"), isFuture)
					}
					break
				}
			}
			if truthy(withoutSuffix) {
				return out
			}
			var pastOrFuture any
			if isFuture {
				pastOrFuture = propOf(loc, "future")
			} else {
				pastOrFuture = propOf(loc, "past")
			}
			if typeOf(pastOrFuture) == "function" {
				return call(pastOrFuture, "pastOrFuture", out)
			}
			s, ok := pastOrFuture.(string)
			if !ok {
				propOf(pastOrFuture, "replace")
				throw("TypeError", "pastOrFuture.replace is not a function")
			}
			return replaceFirst(s, "%s", out)
		}
		fromTo := func(input, withoutSuffix any, instance *Dayjs, isFrom bool) any {
			return fromToBase(input, withoutSuffix, instance, isFrom, Undefined)
		}
		proto["to"] = func(this *Dayjs, a ...any) any { return fromTo(arg(a, 0), arg(a, 1), this, true) }
		proto["from"] = func(this *Dayjs, a ...any) any { return fromTo(arg(a, 0), arg(a, 1), this, false) }
		makeNow := func(d *Dayjs) *Dayjs {
			if truthy(d.u) {
				return staticDayjs("utc")
			}
			return construct(nil)
		}
		proto["toNow"] = func(this *Dayjs, a ...any) any { return this.call("to", makeNow(this), arg(a, 0)) }
		proto["fromNow"] = func(this *Dayjs, a ...any) any { return this.call("from", makeNow(this), arg(a, 0)) }
	})

	definePlugin("localeData", func(o any) {
		getLocalePart := func(part any) any {
			if !truthy(part) {
				return part
			}
			switch part.(type) {
			case []any, []string, string:
				return part
			}
			return propOf(part, "s")
		}
		getShort := func(ins any, target string, full any, num any, localeOrder any) any {
			var locale any
			if truthy(propOf(ins, "name")) {
				locale = ins
			} else {
				locale = ins.(*Dayjs).call("$locale")
			}
			targetLocale := getLocalePart(propOf(locale, target))
			fullLocale := getLocalePart(propOf(locale, toJSString(full)))
			result := targetLocale
			if !truthy(result) {
				result = mapArray(fullLocale, func(f any) any { return jsSliceAny(f, 0.0, num) })
			}
			if !truthy(localeOrder) {
				return result
			}
			weekStart := toNumber(orZero(propOf(locale, "weekStart")))
			arr, ok := toArray(result)
			if !ok {
				propOf(result, "map")
				throw("TypeError", "result.map is not a function")
			}
			out := make([]any, len(arr))
			for i := range arr {
				out[i] = index(result, math.Mod(float64(i)+weekStart, 7))
			}
			return out
		}
		globalLocale := func() any {
			name := toJSString(staticCall("locale"))
			if l, ok := loadedLocale[name]; ok {
				return l
			}
			return Undefined
		}
		longDateFormat := func(l any, format any) any {
			formats := propOf(l, "formats")
			if v := propOf(formats, toJSString(format)); truthy(v) {
				return v
			}
			fs, ok := format.(string)
			if !ok {
				throw("TypeError", "format.toUpperCase is not a function")
			}
			return localizedT(propOf(formats, strings.ToUpper(fs)))
		}
		short := func(this *Dayjs, fmtStr, target string, full any, num any) Func {
			return func(a ...any) any {
				if inst := arg(a, 0); truthy(inst) {
					d, ok := inst.(*Dayjs)
					if !ok {
						throw("TypeError", "instance.format is not a function")
					}
					return d.call("format", fmtStr)
				}
				return getShort(this, target, full, num, Undefined)
			}
		}
		proto["localeData"] = func(this *Dayjs, a ...any) any {
			loc := this.call("$locale")
			return &LocaleData{objFrom(Obj{
				"months", short(this, "MMMM", "months", Undefined, Undefined),
				"monthsShort", short(this, "MMM", "monthsShort", "months", 3.0),
				"firstDayOfWeek", Func(func(a ...any) any { return orZero(propOf(this.call("$locale"), "weekStart")) }),
				"weekdays", short(this, "dddd", "weekdays", Undefined, Undefined),
				"weekdaysMin", short(this, "dd", "weekdaysMin", "weekdays", 2.0),
				"weekdaysShort", short(this, "ddd", "weekdaysShort", "weekdays", 3.0),
				"longDateFormat", Func(func(a ...any) any { return longDateFormat(this.call("$locale"), arg(a, 0)) }),
				"meridiem", propOf(loc, "meridiem"),
				"ordinal", propOf(loc, "ordinal"),
			})}
		}
		statics["months"] = Func(func(a ...any) any { return getShort(globalLocale(), "months", Undefined, Undefined, Undefined) })
		statics["monthsShort"] = Func(func(a ...any) any { return getShort(globalLocale(), "monthsShort", "months", 3.0, Undefined) })
		statics["weekdays"] = Func(func(a ...any) any { return getShort(globalLocale(), "weekdays", nil, nil, arg(a, 0)) })
		statics["weekdaysShort"] = Func(func(a ...any) any { return getShort(globalLocale(), "weekdaysShort", "weekdays", 3.0, arg(a, 0)) })
		statics["weekdaysMin"] = Func(func(a ...any) any { return getShort(globalLocale(), "weekdaysMin", "weekdays", 2.0, arg(a, 0)) })
		statics["localeData"] = Func(func(a ...any) any {
			lo := globalLocale()
			return &LocaleData{objFrom(Obj{
				"firstDayOfWeek", Func(func(a ...any) any { return orZero(propOf(lo, "weekStart")) }),
				"weekdays", Func(func(a ...any) any { return staticCall("weekdays") }),
				"weekdaysShort", Func(func(a ...any) any { return staticCall("weekdaysShort") }),
				"weekdaysMin", Func(func(a ...any) any { return staticCall("weekdaysMin") }),
				"months", Func(func(a ...any) any { return staticCall("months") }),
				"monthsShort", Func(func(a ...any) any { return staticCall("monthsShort") }),
				"longDateFormat", Func(func(a ...any) any { return longDateFormat(lo, arg(a, 0)) }),
				"meridiem", propOf(lo, "meridiem"),
				"ordinal", propOf(lo, "ordinal"),
			})}
		})
	})
}

// LocaleData is the object localeData() returns: functions over a locale's names and formats.
type LocaleData struct{ o *Object }

func (l *LocaleData) fn(name string, args ...any) any { return call(l.o.get(name), name, args...) }

// Months is localeData().months(instance).
func (l *LocaleData) Months(instance ...any) any { return l.fn("months", instance...) }

// MonthsShort is localeData().monthsShort(instance).
func (l *LocaleData) MonthsShort(instance ...any) any { return l.fn("monthsShort", instance...) }

// FirstDayOfWeek is localeData().firstDayOfWeek().
func (l *LocaleData) FirstDayOfWeek() any { return l.fn("firstDayOfWeek") }

// Weekdays is localeData().weekdays(instance).
func (l *LocaleData) Weekdays(instance ...any) any { return l.fn("weekdays", instance...) }

// WeekdaysMin is localeData().weekdaysMin(instance).
func (l *LocaleData) WeekdaysMin(instance ...any) any { return l.fn("weekdaysMin", instance...) }

// WeekdaysShort is localeData().weekdaysShort(instance).
func (l *LocaleData) WeekdaysShort(instance ...any) any { return l.fn("weekdaysShort", instance...) }

// LongDateFormat is localeData().longDateFormat(format).
func (l *LocaleData) LongDateFormat(format any) any { return l.fn("longDateFormat", format) }

// Meridiem is localeData().meridiem: the locale's meridiem function, or Undefined.
func (l *LocaleData) Meridiem() any { return l.o.get("meridiem") }

// Ordinal is localeData().ordinal: the locale's ordinal function, or Undefined.
func (l *LocaleData) Ordinal() any { return l.o.get("ordinal") }
