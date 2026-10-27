package dayjs

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/iamkun/dayjs-go/internal/registry"
)

// Plugin is a Day.js plugin. The plugin package lists all 37; Extend installs one.
type Plugin struct {
	name    string
	install func(option any)
}

// Name is the plugin's Day.js name, such as "customParseFormat".
func (p *Plugin) Name() string { return p.name }

var pluginTable = map[string]*Plugin{}

func definePlugin(name string, install func(option any)) {
	p := &Plugin{name: name, install: install}
	pluginTable[name] = p
	registry.Plugins[name] = p
}

// Extend is dayjs.extend(plugin, option): it installs the plugin once; later calls with the same
// plugin do nothing.
func Extend(p *Plugin, option ...any) {
	if p == nil {
		throw("TypeError", "plugin is not a function")
	}
	if installed[p] {
		return
	}
	p.install(arg(option, 0))
	installed[p] = true
}

// devWarn is console.warn.
func devWarn(msg string) { fmt.Fprintln(os.Stderr, msg) }

// isStartOfArg is `!utils.u(startOf) ? startOf : true` read as a condition.
func isStartOfArg(v any) bool {
	if isUndef(v) {
		return true
	}
	return truthy(v)
}

func construct1(v any) *Dayjs { return construct([]any{v}) }

func staticCall(name string, args ...any) any { return call(statics[name], "dayjs."+name, args...) }

func staticDayjs(name string, args ...any) *Dayjs {
	r := staticCall(name, args...)
	if d, ok := r.(*Dayjs); ok {
		return d
	}
	throw("TypeError", "dayjs."+name+"(...) is not a Day.js object")
	return nil
}

// wrapParse installs a parse method that rewrites cfg.date before the previous parse runs.
func wrapParse(rewrite func(this *Dayjs, cfg *config) any) {
	oldParse := proto["parse"]
	proto["parse"] = func(this *Dayjs, a ...any) any {
		cfg := a[0].(*config)
		cfg.date = rewrite(this, cfg)
		oldParse(this, cfg)
		return Undefined
	}
}

var (
	reAdvanced = regexp.MustCompile(`\[([^\]]+)]|Q|wo|ww|w|WW|W|zzz|z|gggg|GGGG|Do|X|x|k{1,2}|S`)
	reBuddhist = regexp.MustCompile(`(\[[^\]]+])|BBBB|BB`)
)

func init() {
	definePlugin("advancedFormat", func(o any) {
		oldFormat := proto["format"]
		proto["format"] = func(this *Dayjs, a ...any) any {
			locale := this.call("$locale")
			if !truthy(this.call("isValid")) {
				return oldFormat(this, arg(a, 0))
			}
			str := formatDflt
			if f := arg(a, 0); truthy(f) {
				str = toJSString(f)
			}
			ordinal := func(args ...any) any { return call(propOf(locale, "ordinal"), "locale.ordinal", args...) }
			result := replaceAllRegexp(reAdvanced, str, func(match string) string {
				switch match {
				case "Q":
					return toJSString(math.Ceil((this.mo + 1) / 3))
				case "Do":
					return toJSString(ordinal(this.dd))
				case "gggg":
					return toJSString(this.call("weekYear"))
				case "GGGG":
					return toJSString(this.call("isoWeekYear"))
				case "wo":
					return toJSString(ordinal(this.call("week"), "W"))
				case "w", "ww":
					n := 1.0
					if match == "ww" {
						n = 2
					}
					return toJSString(padStart(this.call("week"), n, "0"))
				case "W", "WW":
					n := 1.0
					if match == "WW" {
						n = 2
					}
					return toJSString(padStart(this.call("isoWeek"), n, "0"))
				case "k", "kk":
					n := 1.0
					if match == "kk" {
						n = 2
					}
					h := this.h
					if h == 0 {
						h = 24
					}
					return toJSString(padStart(numberToString(h), n, "0"))
				case "X":
					return toJSString(math.Floor(this.d.GetTime() / 1000))
				case "x":
					return toJSString(this.d.GetTime())
				case "z":
					return "[" + toJSString(this.call("offsetName")) + "]"
				case "zzz":
					return "[" + toJSString(this.call("offsetName", "long")) + "]"
				}
				return match
			})
			return oldFormat(this, result)
		}
	})

	definePlugin("arraySupport", func(o any) {
		wrapParse(func(this *Dayjs, cfg *config) any {
			arr, ok := cfg.date.([]any)
			if !ok {
				return cfg.date
			}
			if truthy(cfg.utc) {
				if len(arr) == 0 {
					return NewDate()
				}
				return &Date{timeClip(dateUTC(arr...))}
			}
			if len(arr) == 1 {
				return construct1(toJSString(arr[0])).call("toDate")
			}
			return NewDate(arr...)
		})
	})

	definePlugin("badMutable", func(o any) {
		proto["$g"] = func(this *Dayjs, a ...any) any {
			if isUndef(arg(a, 0)) {
				return this.field(toJSString(arg(a, 1)))
			}
			return this.call("$set", arg(a, 2), arg(a, 0))
		}
		proto["set"] = func(this *Dayjs, a ...any) any { return this.call("$set", arg(a, 0), arg(a, 1)) }
		oldStartOf := proto["startOf"]
		proto["startOf"] = func(this *Dayjs, a ...any) any {
			r := oldStartOf(this, arg(a, 0), arg(a, 1)).(*Dayjs)
			this.d = r.call("toDate").(*Date)
			this.call("init")
			return this
		}
		oldAdd := proto["add"]
		proto["add"] = func(this *Dayjs, a ...any) any {
			r := oldAdd(this, arg(a, 0), arg(a, 1)).(*Dayjs)
			this.d = r.call("toDate").(*Date)
			this.call("init")
			return this
		}
		oldLocale := proto["locale"]
		proto["locale"] = func(this *Dayjs, a ...any) any {
			if !truthy(arg(a, 0)) {
				return this.l
			}
			this.l = oldLocale(this, arg(a, 0), arg(a, 1)).(*Dayjs).l
			return this
		}
		oldDaysInMonth := proto["daysInMonth"]
		proto["daysInMonth"] = func(this *Dayjs, a ...any) any { return oldDaysInMonth(this.callDayjs("clone")) }
		for _, name := range []string{"isSame", "isBefore", "isAfter"} {
			old := proto[name]
			proto[name] = func(this *Dayjs, a ...any) any { return old(this.callDayjs("clone"), arg(a, 0), arg(a, 1)) }
		}
	})

	definePlugin("bigIntSupport", func(o any) {
		wrapParse(func(this *Dayjs, cfg *config) any {
			if typeOf(cfg.date) == "bigint" {
				return toNumber(cfg.date)
			}
			return cfg.date
		})
		oldUnix := statics["unix"]
		statics["unix"] = Func(func(a ...any) any {
			ts := arg(a, 0)
			if typeOf(ts) == "bigint" {
				ts = toNumber(ts)
			}
			return call(oldUnix, "oldUnix", ts)
		})
	})

	definePlugin("buddhistEra", func(o any) {
		oldFormat := proto["format"]
		proto["format"] = func(this *Dayjs, a ...any) any {
			str := formatDflt
			if f := arg(a, 0); truthy(f) {
				str = toJSString(f)
			}
			result := replaceAllSubmatchFunc(reBuddhist, str, func(m []string, has []bool) string {
				year := numberToString(this.y + 543)
				var v any
				if m[0] == "BB" {
					v = padStart(jsSlice(year, -2.0, Undefined), 2, "0")
				} else {
					v = padStart(year, 4, "0")
				}
				if has[1] && m[1] != "" {
					return m[1]
				}
				return toJSString(v)
			})
			return oldFormat(this, result)
		}
	})

	definePlugin("calendar", func(o any) {
		lt, l := "h:mm A", "MM/DD/YYYY"
		calendarFormat := objFrom(Obj{
			"lastDay", "[Yesterday at] " + lt,
			"sameDay", "[Today at] " + lt,
			"nextDay", "[Tomorrow at] " + lt,
			"nextWeek", "dddd [at] " + lt,
			"lastWeek", "[Last] dddd [at] " + lt,
			"sameElse", l,
		})
		proto["calendar"] = func(this *Dayjs, a ...any) any {
			referenceTime, formats := arg(a, 0), arg(a, 1)
			format := formats
			if !truthy(format) {
				format = this.locProp("calendar")
				if !truthy(format) {
					format = calendarFormat
				}
			}
			ref := referenceTime
			if !truthy(ref) {
				ref = Undefined
			}
			refStart := construct1(ref).callDayjs("startOf", "d")
			diff := toNumber(this.call("diff", refStart, "d", true))
			var retVal string
			switch {
			case diff < -6:
				retVal = "sameElse"
			case diff < -1:
				retVal = "lastWeek"
			case diff < 0:
				retVal = "lastDay"
			case diff < 1:
				retVal = "sameDay"
			case diff < 2:
				retVal = "nextDay"
			case diff < 7:
				retVal = "nextWeek"
			default:
				retVal = "sameElse"
			}
			current := propOf(format, retVal)
			if !truthy(current) {
				current = calendarFormat.get(retVal)
			}
			if typeOf(current) == "function" {
				return call(current, "currentFormat", construct(nil))
			}
			return this.call("format", current)
		}
	})

	definePlugin("dayOfYear", func(o any) {
		proto["dayOfYear"] = func(this *Dayjs, a ...any) any {
			start := toNumber(construct1(this).call("startOf", "day"))
			year := toNumber(construct1(this).call("startOf", "year"))
			doy := mathRound((start-year)/864e5) + 1
			if isNullish(arg(a, 0)) {
				return doy
			}
			return this.call("add", toNumber(arg(a, 0))-doy, "day")
		}
	})

	definePlugin("devHelper", func(o any) {
		oldParse := proto["parse"]
		proto["parse"] = func(this *Dayjs, a ...any) any {
			cfg := a[0].(*config)
			date := cfg.date
			if s, ok := date.(string); ok && utf16Len(s) == 13 {
				devWarn("To parse a Unix timestamp like " + s + ", you should pass it as a Number. https://day.js.org/docs/en/parse/unix-timestamp-milliseconds")
			}
			if n, ok := date.(float64); ok && utf16Len(numberToString(n)) == 4 {
				s := numberToString(n)
				devWarn("Guessing you may want to parse the Year " + s + ", you should pass it as a String " + s + ", not a Number. Otherwise, " + s + " will be treated as a Unix timestamp")
			}
			if len(cfg.args) >= 2 && !customParse {
				devWarn("To parse a date-time string like " + toJSString(date) + " using the given format, you should enable customParseFormat plugin first. https://day.js.org/docs/en/parse/string-format")
			}
			return oldParse(this, cfg)
		}
		oldLocale := statics["locale"]
		statics["locale"] = Func(func(a ...any) any {
			preset, object, isLocal := arg(a, 0), arg(a, 1), arg(a, 2)
			if s, ok := preset.(string); ok && isUndef(object) {
				if _, ok := loadedLocale[s]; !ok {
					devWarn("Guessing you may want to use locale " + s + ", you have to load it before using it. https://day.js.org/docs/en/i18n/loading-into-nodejs")
				}
			}
			return call(oldLocale, "oldLocale", preset, object, isLocal)
		})
		oldDiff := proto["diff"]
		proto["diff"] = func(this *Dayjs, a ...any) any {
			date := arg(a, 0)
			if !truthy(date) || !truthy(construct1(date).call("isValid")) {
				devWarn("Invalid usage: diff() requires a valid comparison date as the first argument. https://day.js.org/docs/en/display/difference")
			}
			return oldDiff(this, date, arg(a, 1), arg(a, 2))
		}
	})

	definePlugin("isBetween", func(o any) {
		proto["isBetween"] = func(this *Dayjs, a ...any) any {
			dA, dB := construct1(arg(a, 0)), construct1(arg(a, 1))
			u, i := arg(a, 2), arg(a, 3)
			if !truthy(i) {
				i = "()"
			}
			dAi := strictEq(index(i, 0), "(")
			dBi := strictEq(index(i, 1), ")")
			is := func(name string, d *Dayjs) bool { return truthy(this.call(name, d, u)) }
			left := func() bool {
				if dAi {
					return is("isAfter", dA)
				}
				return !is("isBefore", dA)
			}
			right := func() bool {
				if dBi {
					return is("isBefore", dB)
				}
				return !is("isAfter", dB)
			}
			left2 := func() bool {
				if dAi {
					return is("isBefore", dA)
				}
				return !is("isAfter", dA)
			}
			right2 := func() bool {
				if dBi {
					return is("isAfter", dB)
				}
				return !is("isBefore", dB)
			}
			return (left() && right()) || (left2() && right2())
		}
	})

	definePlugin("isLeapYear", func(o any) {
		proto["isLeapYear"] = func(this *Dayjs, a ...any) any {
			y := this.y
			return (math.Mod(y, 4) == 0 && math.Mod(y, 100) != 0) || math.Mod(y, 400) == 0
		}
	})

	definePlugin("isMoment", func(o any) {
		statics["isMoment"] = Func(func(a ...any) any { return isDayjs(arg(a, 0)) })
	})

	definePlugin("isSameOrAfter", func(o any) {
		proto["isSameOrAfter"] = func(this *Dayjs, a ...any) any {
			return truthy(this.call("isSame", arg(a, 0), arg(a, 1))) || truthy(this.call("isAfter", arg(a, 0), arg(a, 1)))
		}
	})

	definePlugin("isSameOrBefore", func(o any) {
		proto["isSameOrBefore"] = func(this *Dayjs, a ...any) any {
			return truthy(this.call("isSame", arg(a, 0), arg(a, 1))) || truthy(this.call("isBefore", arg(a, 0), arg(a, 1)))
		}
	})

	sameDay := func(this *Dayjs, other *Dayjs) bool {
		return strictEq(this.call("format", "YYYY-MM-DD"), other.call("format", "YYYY-MM-DD"))
	}
	definePlugin("isToday", func(o any) {
		proto["isToday"] = func(this *Dayjs, a ...any) any { return sameDay(this, construct(nil)) }
	})
	definePlugin("isTomorrow", func(o any) {
		proto["isTomorrow"] = func(this *Dayjs, a ...any) any {
			return sameDay(this, construct(nil).callDayjs("add", 1.0, "day"))
		}
	})
	definePlugin("isYesterday", func(o any) {
		proto["isYesterday"] = func(this *Dayjs, a ...any) any {
			return sameDay(this, construct(nil).callDayjs("subtract", 1.0, "day"))
		}
	})

	definePlugin("isoWeek", func(o any) {
		getYearFirstThursday := func(year any, isUtc any) *Dayjs {
			var base *Dayjs
			if truthy(isUtc) {
				base = staticDayjs("utc")
			} else {
				base = construct(nil)
			}
			first := base.callDayjs("year", year).callDayjs("startOf", unitY)
			add := 4 - toNumber(first.call("isoWeekday"))
			if toNumber(first.call("isoWeekday")) > 4 {
				add += 7
			}
			return first.callDayjs("add", add, unitD)
		}
		currentWeekThursday := func(ins *Dayjs) *Dayjs {
			return ins.callDayjs("add", 4-toNumber(ins.call("isoWeekday")), unitD)
		}
		proto["isoWeekYear"] = func(this *Dayjs, a ...any) any { return currentWeekThursday(this).call("year") }
		proto["isoWeek"] = func(this *Dayjs, a ...any) any {
			if week := arg(a, 0); !isUndef(week) {
				return this.call("add", (toNumber(week)-toNumber(this.call("isoWeek")))*7, unitD)
			}
			now := currentWeekThursday(this)
			first := getYearFirstThursday(this.call("isoWeekYear"), this.u)
			return toNumber(now.call("diff", first, unitW)) + 1
		}
		proto["isoWeekday"] = func(this *Dayjs, a ...any) any {
			if week := arg(a, 0); !isUndef(week) {
				if math.Mod(toNumber(this.call("day")), 7) != 0 {
					return this.call("day", week)
				}
				return this.call("day", toNumber(week)-7)
			}
			d := this.call("day")
			if !truthy(d) {
				return 7.0
			}
			return d
		}
		oldStartOf := proto["startOf"]
		proto["startOf"] = func(this *Dayjs, a ...any) any {
			isStartOf := isStartOfArg(arg(a, 1))
			if prettyUnit(arg(a, 0)) == "isoweek" {
				date, wd := toNumber(this.call("date")), toNumber(this.call("isoWeekday"))
				if isStartOf {
					return this.callDayjs("date", date-(wd-1)).call("startOf", "day")
				}
				return this.callDayjs("date", (date-1-(wd-1))+7).call("endOf", "day")
			}
			return oldStartOf(this, arg(a, 0), arg(a, 1))
		}
	})

	definePlugin("isoWeeksInYear", func(o any) {
		proto["isoWeeksInYear"] = func(this *Dayjs, a ...any) any {
			isLeap := truthy(this.call("isLeapYear"))
			day := this.callDayjs("endOf", "y").call("day")
			if strictEq(day, 4.0) || (isLeap && strictEq(day, 5.0)) {
				return 53.0
			}
			return 52.0
		}
	})

	definePlugin("minMax", func(o any) {
		sortBy := func(method string, dates []any) any {
			if len(dates) == 0 || (len(dates) == 1 && !truthy(dates[0])) {
				return nil
			}
			if len(dates) == 1 {
				if arr, ok := dates[0].([]any); ok {
					if len(arr) == 0 {
						return nil
					}
					dates = arr
				}
			}
			var kept []any
			for _, d := range dates {
				if truthy(d) {
					kept = append(kept, d)
				}
			}
			if len(kept) == 0 {
				return Undefined
			}
			result := kept[0]
			for _, v := range kept[1:] {
				d, ok := v.(*Dayjs)
				if !ok {
					throw("TypeError", "dates[i].isValid is not a function")
				}
				if !truthy(d.call("isValid")) || truthy(d.call(method, result)) {
					result = d
				}
			}
			return result
		}
		statics["max"] = Func(func(a ...any) any { return sortBy("isAfter", a) })
		statics["min"] = Func(func(a ...any) any { return sortBy("isBefore", a) })
	})

	definePlugin("negativeYear", func(o any) {
		wrapParse(func(this *Dayjs, cfg *config) any {
			s, ok := cfg.date.(string)
			if !ok || !strings.HasPrefix(s, "-") {
				return cfg.date
			}
			normal := jsSlice(s, 1.0, Undefined)
			newDate := construct1(normal)
			if truthy(cfg.utc) {
				newDate = staticDayjs("utc", normal)
			} else {
				newDate = construct1(normal)
			}
			fullYear := newDate.call("year")
			if strings.Contains(s, "-"+toJSString(fullYear)) {
				return construct1(newDate).callDayjs("subtract", toNumber(fullYear)*2, "year").call("toDate")
			}
			return s
		})
	})

	definePlugin("objectSupport", func(o any) {
		isObject := func(v any) bool {
			switch v.(type) {
			case Obj, *Object:
				return true
			}
			return false
		}
		objPretty := func(u any) string {
			unit := prettyUnit(u)
			if unit == "date" {
				return "day"
			}
			return unit
		}
		wrapParse(func(this *Dayjs, cfg *config) any {
			date := cfg.date
			if !isObject(date) {
				return date
			}
			keys := objectKeys(date)
			if len(keys) == 0 {
				return NewDate()
			}
			var now *Dayjs
			if truthy(cfg.utc) {
				now = staticDayjs("utc")
			} else {
				now = construct(nil)
			}
			d := newObj()
			for _, k := range keys {
				d.set(objPretty(k), propOf(date, k))
			}
			day, year, month := d.get("day"), d.get("year"), d.get("month")
			monthSet := toNumber(month) >= 0
			var dd any = day
			if !truthy(dd) {
				if !truthy(year) && !monthSet {
					dd = now.call("date")
				} else {
					dd = 1.0
				}
			}
			var y any = year
			if !truthy(y) {
				y = now.call("year")
			}
			var M any
			switch {
			case monthSet:
				M = month
			case !truthy(year) && !truthy(day):
				M = now.call("month")
			default:
				M = 0.0
			}
			h, m, s, ms := orZero(d.get("hour")), orZero(d.get("minute")), orZero(d.get("second")), orZero(d.get("millisecond"))
			if truthy(cfg.utc) {
				return &Date{timeClip(dateUTC(y, M, dd, h, m, s, ms))}
			}
			return NewDate(y, M, dd, h, m, s, ms)
		})
		oldSet, oldAdd, oldSubtract := proto["set"], proto["add"], proto["subtract"]
		callObject := func(this *Dayjs, fn func(chain *Dayjs, v float64, key string) any, argument any, offset float64) any {
			var chain any = this
			for _, k := range objectKeys(argument) {
				c, ok := chain.(*Dayjs)
				if !ok {
					throw("TypeError", "chain is not a Day.js object")
				}
				chain = fn(c, toNumber(propOf(argument, k))*offset, k)
			}
			return chain
		}
		proto["set"] = func(this *Dayjs, a ...any) any {
			unit, value := arg(a, 0), arg(a, 1)
			if isUndef(value) {
				value = unit
			}
			if isPlainObject(unit) {
				return callObject(this, func(c *Dayjs, v float64, k string) any { return oldSet(c, k, v) }, value, 1)
			}
			return oldSet(this, unit, value)
		}
		proto["add"] = func(this *Dayjs, a ...any) any {
			if isPlainObject(arg(a, 0)) {
				return callObject(this, func(c *Dayjs, v float64, k string) any { return oldAdd(c, v, k) }, arg(a, 0), 1)
			}
			return oldAdd(this, arg(a, 0), arg(a, 1))
		}
		proto["subtract"] = func(this *Dayjs, a ...any) any {
			if isPlainObject(arg(a, 0)) {
				return callObject(this, func(c *Dayjs, v float64, k string) any { return oldAdd(c, v, k) }, arg(a, 0), -1)
			}
			return oldSubtract(this, arg(a, 0), arg(a, 1))
		}
	})

	definePlugin("pluralGetSet", func(o any) {
		for _, alias := range []string{"milliseconds", "seconds", "minutes", "hours", "days", "weeks",
			"isoWeeks", "months", "quarters", "years", "dates"} {
			proto[alias] = proto[strings.TrimSuffix(alias, "s")]
		}
	})

	definePlugin("quarterOfYear", func(o any) {
		proto["quarter"] = func(this *Dayjs, a ...any) any {
			if q := arg(a, 0); !isUndef(q) {
				return this.call("month", math.Mod(toNumber(this.call("month")), 3)+(toNumber(q)-1)*3)
			}
			return math.Ceil((toNumber(this.call("month")) + 1) / 3)
		}
		oldAdd := proto["add"]
		proto["add"] = func(this *Dayjs, a ...any) any {
			number := toNumber(arg(a, 0))
			if prettyUnit(arg(a, 1)) == unitQ {
				return this.call("add", number*3, unitM)
			}
			return oldAdd(this, number, arg(a, 1))
		}
		oldStartOf := proto["startOf"]
		proto["startOf"] = func(this *Dayjs, a ...any) any {
			isStartOf := isStartOfArg(arg(a, 1))
			if prettyUnit(arg(a, 0)) == unitQ {
				quarter := toNumber(this.call("quarter")) - 1
				if isStartOf {
					return this.callDayjs("month", quarter*3).callDayjs("startOf", unitM).call("startOf", unitD)
				}
				return this.callDayjs("month", quarter*3+2).callDayjs("endOf", unitM).call("endOf", unitD)
			}
			return oldStartOf(this, arg(a, 0), arg(a, 1))
		}
	})

	definePlugin("toArray", func(o any) {
		proto["toArray"] = func(this *Dayjs, a ...any) any {
			return []any{this.field("$y"), this.field("$M"), this.field("$D"), this.field("$H"),
				this.field("$m"), this.field("$s"), this.field("$ms")}
		}
	})

	definePlugin("toObject", func(o any) {
		proto["toObject"] = func(this *Dayjs, a ...any) any {
			return objFrom(Obj{"years", this.field("$y"), "months", this.field("$M"), "date", this.field("$D"),
				"hours", this.field("$H"), "minutes", this.field("$m"), "seconds", this.field("$s"),
				"milliseconds", this.field("$ms")})
		}
	})

	definePlugin("updateLocale", func(o any) {
		statics["updateLocale"] = Func(func(a ...any) any {
			lc, ok := loadedLocale[toJSString(arg(a, 0))]
			if !ok || lc == nil {
				return Undefined
			}
			custom := arg(a, 1)
			var keys []string
			if truthy(custom) {
				keys = objectKeys(custom)
			}
			for _, c := range keys {
				cur, nv := lc.get(c), propOf(custom, c)
				_, curIsArray := toArray(cur)
				if truthy(cur) && truthy(nv) && typeOf(cur) == "object" && typeOf(nv) == "object" && !curIsArray {
					merged := newObj()
					spreadInto(merged, cur)
					spreadInto(merged, nv)
					lc.set(c, merged)
				} else {
					lc.set(c, nv)
				}
			}
			return lc
		})
	})

	definePlugin("weekOfYear", func(o any) {
		proto["week"] = func(this *Dayjs, a ...any) any {
			week := arg(a, 0)
			if isUndef(week) {
				week = nil
			}
			if week != nil {
				return this.call("add", (toNumber(week)-toNumber(this.call("week")))*7, unitD)
			}
			yearStart := this.locProp("yearStart")
			if !truthy(yearStart) {
				yearStart = 1.0
			}
			if strictEq(this.call("month"), 11.0) && toNumber(this.call("date")) > 25 {
				next := construct1(this).callDayjs("startOf", unitY).callDayjs("add", 1.0, unitY).callDayjs("date", yearStart)
				endOfWeek := construct1(this).callDayjs("endOf", unitW)
				if truthy(next.call("isBefore", endOfWeek)) {
					return 1.0
				}
			}
			yearStartDay := construct1(this).callDayjs("startOf", unitY).callDayjs("date", yearStart)
			yearStartWeek := yearStartDay.callDayjs("startOf", unitW).callDayjs("subtract", 1.0, unitMS)
			diffInWeek := toNumber(this.call("diff", yearStartWeek, unitW, true))
			if diffInWeek < 0 {
				return construct1(this).callDayjs("startOf", "week").call("week")
			}
			return math.Ceil(diffInWeek)
		}
		proto["weeks"] = func(this *Dayjs, a ...any) any {
			week := arg(a, 0)
			if isUndef(week) {
				week = nil
			}
			return this.call("week", week)
		}
	})

	definePlugin("weekYear", func(o any) {
		proto["weekYear"] = func(this *Dayjs, a ...any) any {
			month, week, year := this.call("month"), this.call("week"), this.call("year")
			if strictEq(week, 1.0) && strictEq(month, 11.0) {
				return toNumber(year) + 1
			}
			if strictEq(month, 0.0) && toNumber(week) >= 52 {
				return toNumber(year) - 1
			}
			return year
		}
	})

	definePlugin("weekday", func(o any) {
		proto["weekday"] = func(this *Dayjs, a ...any) any {
			weekStart := toNumber(orZero(this.locProp("weekStart")))
			w := this.w
			if w < weekStart {
				w += 7
			}
			weekday := w - weekStart
			if isUndef(arg(a, 0)) {
				return weekday
			}
			return this.callDayjs("subtract", weekday, "day").call("add", arg(a, 0), "day")
		}
	})

	definePlugin("preParsePostFormat", func(o any) {
		oldParse := proto["parse"]
		proto["parse"] = func(this *Dayjs, a ...any) any {
			cfg := a[0].(*config)
			if s, ok := cfg.date.(string); ok {
				loc := this.call("$locale")
				if truthy(loc) && truthy(propOf(loc, "preparse")) {
					cfg.date = call(propOf(loc, "preparse"), "locale.preparse", s)
				}
			}
			return oldParse(this, cfg)
		}
		oldFormat := proto["format"]
		proto["format"] = func(this *Dayjs, a ...any) any {
			result := oldFormat(this, a...)
			loc := this.call("$locale")
			if truthy(loc) && truthy(propOf(loc, "postformat")) {
				return call(propOf(loc, "postformat"), "locale.postformat", result)
			}
			return result
		}
		if old := fromToBase; old != nil {
			fromToBase = func(input, withoutSuffix any, instance *Dayjs, isFrom bool, _ any) any {
				var loc any = Undefined
				if l, ok := loadedLocale["undefined"]; ok {
					loc = l
				}
				if !truthy(loc) {
					loc = instance.call("$locale")
				}
				var postformat any = Undefined
				if truthy(loc) {
					postformat = propOf(loc, "postformat")
				}
				return old(input, withoutSuffix, instance, isFrom, postformat)
			}
		}
	})
}

// spreadInto copies an object's own enumerable properties, as {...src} does.
func spreadInto(dst *Object, src any) {
	switch s := src.(type) {
	case *Object:
		for _, k := range s.keys {
			dst.set(k, s.get(k))
		}
	case Obj:
		o := objFrom(s)
		for _, k := range o.keys {
			dst.set(k, o.get(k))
		}
	case []any:
		for i, v := range s {
			dst.set(numberToString(float64(i)), v)
		}
	case *Function:
		if s.Props != nil {
			spreadInto(dst, s.Props)
		}
	}
}
