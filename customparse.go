package dayjs

import (
	"math"
	"regexp"
)

var (
	cpfTokens    = jsRegexp(`(\[[^[]*\])|([-_:/.,()\s]+)|(A|a|Q|YYYY|YY?|ww?|MM?M?M?|Do|DD?|hh?|HH?|mm?|ss?|S{1,3}|z|ZZ?)`)
	cpfMatch1    = regexp.MustCompile(`\d`)
	cpfMatch2    = regexp.MustCompile(`\d\d`)
	cpfMatch3    = regexp.MustCompile(`\d{3}`)
	cpfMatch4    = regexp.MustCompile(`\d{4}`)
	cpfMatch1to2 = regexp.MustCompile(`\d\d?`)
	cpfSigned    = regexp.MustCompile(`[+-]?\d+`)
	cpfOffset    = regexp.MustCompile(`[+-]\d\d:?(\d\d)?|Z`)
	cpfWord      = jsRegexp(`\d*[^-_:/,()\s\d]+`)
	cpfDigits    = regexp.MustCompile(`\d+`)
	cpfBrackets  = regexp.MustCompile(`\[|\]`)
	cpfEdgeBrkt  = regexp.MustCompile(`^\[|\]$`)

	// cpfLocale is the plugin's module-level locale variable.
	cpfLocale any = newObj()
	// parseTwoDigitYear is the plugin's module-level parseTwoDigitYear, replaceable by the option.
	parseTwoDigitYear any = Func(func(a ...any) any {
		input := toNumber(arg(a, 0))
		if input > 68 {
			return input + 1900
		}
		return input + 2000
	})
)

type cpfExpr struct {
	re     *regexp.Regexp
	parser func(time *Object, input string)
}

func cpfOffsetFromString(s string) float64 {
	if s == "" || s == "Z" {
		return 0
	}
	parts := reOffsetHM.FindAllString(s, -1)
	part := func(i int) any {
		if i < len(parts) {
			return parts[i]
		}
		return Undefined
	}
	extra := toNumber(part(2))
	if extra == 0 || math.IsNaN(extra) {
		extra = 0
	}
	minutes := toNumber(part(1))*60 + extra
	if minutes == 0 {
		return 0
	}
	if strictEq(part(0), "+") {
		return -minutes
	}
	return minutes
}

func cpfLocalePart(name string) any {
	part := propOf(cpfLocale, name)
	if !truthy(part) {
		return part
	}
	switch part.(type) {
	case []any, []string, string:
		return part
	}
	s, f := propOf(part, "s"), propOf(part, "f")
	sa, ok := toArray(s)
	if !ok {
		propOf(s, "concat")
		throw("TypeError", "part.s.concat is not a function")
	}
	out := append([]any(nil), sa...)
	if fa, ok := toArray(f); ok {
		return append(out, fa...)
	}
	return append(out, f)
}

func meridiemMatch(input string, isLowerCase bool) any {
	meridiem := propOf(cpfLocale, "meridiem")
	if !truthy(meridiem) {
		if isLowerCase {
			return input == "pm"
		}
		return input == "PM"
	}
	for i := 1; i <= 24; i++ {
		m := toJSString(call(meridiem, "meridiem", float64(i), 0.0, isLowerCase))
		if indexOf16(input, m) > -1 {
			return i > 12
		}
	}
	return Undefined
}

// indexOf16 is String.prototype.indexOf: the UTF-16 index of the first occurrence, or -1.
func indexOf16(s, sub string) int {
	i := -1
	for j := 0; j+len(sub) <= len(s); j++ {
		if s[j:j+len(sub)] == sub {
			i = j
			break
		}
	}
	if i < 0 {
		return -1
	}
	return utf16Len(s[:i])
}

func addInput(property string) func(*Object, string) {
	return func(t *Object, input string) { t.set(property, stringToNumber(input)) }
}

var cpfExpressions map[string]cpfExpr

func init() {
	zone := cpfExpr{cpfOffset, func(t *Object, input string) {
		z, ok := t.get("zone").(*Object)
		if !ok {
			z = newObj()
			t.set("zone", z)
		}
		z.set("offset", cpfOffsetFromString(input))
	}}
	monthIndex := func(list any, input string) float64 {
		arr, ok := toArray(list)
		if !ok {
			propOf(list, "indexOf")
			throw("TypeError", "indexOf is not a function")
		}
		i := float64(arrayIndexOf(arr, input) + 1)
		if i < 1 {
			throw("Error", "")
		}
		if m := math.Mod(i, 12); m != 0 {
			return m
		}
		return i
	}
	cpfExpressions = map[string]cpfExpr{
		"A":   {cpfWord, func(t *Object, input string) { t.set("afternoon", meridiemMatch(input, false)) }},
		"a":   {cpfWord, func(t *Object, input string) { t.set("afternoon", meridiemMatch(input, true)) }},
		"Q":   {cpfMatch1, func(t *Object, input string) { t.set("month", (stringToNumber(input)-1)*3+1) }},
		"S":   {cpfMatch1, func(t *Object, input string) { t.set("milliseconds", stringToNumber(input)*100) }},
		"SS":  {cpfMatch2, func(t *Object, input string) { t.set("milliseconds", stringToNumber(input)*10) }},
		"SSS": {cpfMatch3, func(t *Object, input string) { t.set("milliseconds", stringToNumber(input)) }},
		"s":   {cpfMatch1to2, addInput("seconds")},
		"ss":  {cpfMatch1to2, addInput("seconds")},
		"m":   {cpfMatch1to2, addInput("minutes")},
		"mm":  {cpfMatch1to2, addInput("minutes")},
		"H":   {cpfMatch1to2, addInput("hours")},
		"h":   {cpfMatch1to2, addInput("hours")},
		"HH":  {cpfMatch1to2, addInput("hours")},
		"hh":  {cpfMatch1to2, addInput("hours")},
		"D":   {cpfMatch1to2, addInput("day")},
		"DD":  {cpfMatch2, addInput("day")},
		"Do": {cpfWord, func(t *Object, input string) {
			ordinal := propOf(cpfLocale, "ordinal")
			m := cpfDigits.FindString(input)
			if m == "" {
				throw("TypeError", "input.match(...) is not iterable")
			}
			t.set("day", m)
			if !truthy(ordinal) {
				return
			}
			for i := 1; i <= 31; i++ {
				r := call(ordinal, "ordinal", float64(i))
				s, ok := r.(string)
				if !ok {
					throw("TypeError", "ordinal(...).replace is not a function")
				}
				if cpfBrackets.ReplaceAllString(s, "") == input {
					t.set("day", float64(i))
				}
			}
		}},
		"w":  {cpfMatch1to2, addInput("week")},
		"ww": {cpfMatch2, addInput("week")},
		"M":  {cpfMatch1to2, addInput("month")},
		"MM": {cpfMatch2, addInput("month")},
		"MMM": {cpfWord, func(t *Object, input string) {
			months := cpfLocalePart("months")
			monthsShort := cpfLocalePart("monthsShort")
			list := monthsShort
			if !truthy(list) {
				list = mapArray(months, func(m any) any { return jsSliceAny(m, 0.0, 3.0) })
			}
			t.set("month", monthIndex(list, input))
		}},
		"MMMM": {cpfWord, func(t *Object, input string) {
			t.set("month", monthIndex(cpfLocalePart("months"), input))
		}},
		"Y":    {cpfSigned, addInput("year")},
		"YY":   {cpfMatch2, func(t *Object, input string) { t.set("year", call(parseTwoDigitYear, "parseTwoDigitYear", input)) }},
		"YYYY": {cpfMatch4, addInput("year")},
		"Z":    zone,
		"ZZ":   zone,
	}

	definePlugin("customParseFormat", func(o any) {
		customParse = true
		if truthy(o) {
			if f := propOf(o, "parseTwoDigitYear"); truthy(f) {
				parseTwoDigitYear = f
			}
		}
		oldParse := proto["parse"]
		proto["parse"] = func(this *Dayjs, a ...any) any {
			cfg := a[0].(*config)
			date, utc, args := cfg.date, cfg.utc, cfg.args
			this.u = utc
			format := arg(args, 1)
			if fs, ok := format.(string); ok {
				isStrictWithoutLocale := strictEq(arg(args, 2), true)
				isStrictWithLocale := strictEq(arg(args, 3), true)
				isStrict := isStrictWithoutLocale || isStrictWithLocale
				pl := arg(args, 2)
				cpfLocale = this.call("$locale")
				if !isStrictWithoutLocale && truthy(pl) {
					if l, ok := loadedLocale[toJSString(pl)]; ok {
						cpfLocale = l
					} else {
						cpfLocale = Undefined
					}
				}
				this.d = parseFormattedInput(date, fs, utc)
				this.call("init")
				if truthy(pl) && !strictEq(pl, true) {
					this.l = this.callDayjs("locale", pl).l
				}
				if isStrict && !looseEqString(date, toJSString(this.call("format", fs))) {
					this.d = &Date{math.NaN()}
				}
				cpfLocale = newObj()
			} else if list, ok := format.([]any); ok {
				n := len(list)
				for i := 1; i <= n; i++ {
					args[1] = list[i-1]
					result := construct(args)
					if truthy(result.call("isValid")) {
						this.d = result.d
						this.l = result.l
						this.call("init")
						break
					}
					if i == n {
						this.d = &Date{math.NaN()}
					}
				}
			} else {
				oldParse(this, cfg)
			}
			return Undefined
		}
	})
}

type cpfToken struct {
	literal string
	isLit   bool
	expr    cpfExpr
}

func makeParser(format string) func(input any) *Object {
	var formats any = Undefined
	if truthy(cpfLocale) {
		formats = propOf(cpfLocale, "formats")
	}
	format = localizedU(format, formats)
	matches := cpfTokens.FindAllString(format, -1)
	if matches == nil {
		throw("TypeError", "Cannot destructure property 'length' of 'array' as it is null.")
	}
	tokens := make([]cpfToken, len(matches))
	for i, tok := range matches {
		if e, ok := cpfExpressions[tok]; ok {
			tokens[i] = cpfToken{expr: e}
		} else {
			tokens[i] = cpfToken{literal: cpfEdgeBrkt.ReplaceAllString(tok, ""), isLit: true}
		}
	}
	return func(input any) *Object {
		time := newObj()
		start := 0
		for _, tok := range tokens {
			if tok.isLit {
				start += utf16Len(tok.literal)
				continue
			}
			s, ok := input.(string)
			if !ok {
				throw("TypeError", "input.slice is not a function")
			}
			part := jsSlice(s, float64(start), Undefined)
			loc := tok.expr.re.FindStringIndex(part)
			if loc == nil {
				throw("TypeError", "Cannot read properties of null (reading '0')")
			}
			value := part[loc[0]:loc[1]]
			tok.expr.parser(time, value)
			input = replaceFirst(s, value, "")
		}
		correctHours(time)
		return time
	}
}

func correctHours(time *Object) {
	afternoon := time.get("afternoon")
	if isUndef(afternoon) {
		return
	}
	hours := time.get("hours")
	if truthy(afternoon) {
		if toNumber(hours) < 12 {
			time.set("hours", jsAdd(hours, 12.0))
		}
	} else if strictEq(hours, 12.0) {
		time.set("hours", 0.0)
	}
	time.del("afternoon")
}

// parseFormattedInput is the plugin's parseFormattedInput: any exception yields an invalid date.
func parseFormattedInput(input any, format string, utc any) (out *Date) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*Error); !ok {
				panic(r)
			}
			out = &Date{math.NaN()}
		}
	}()
	if format == "x" || format == "X" {
		mul := 1.0
		if format == "X" {
			mul = 1000
		}
		return &Date{timeClip(mul * toNumber(input))}
	}
	parser := makeParser(format)
	t := parser(input)
	year, month, day := t.get("year"), t.get("month"), t.get("day")
	now := NewDate()
	var d any = day
	if !truthy(d) {
		if !truthy(year) && !truthy(month) {
			d = now.getDate()
		} else {
			d = 1.0
		}
	}
	var y any = year
	if !truthy(y) {
		y = now.getFullYear()
	}
	var M any = 0.0
	if !(truthy(year) && !truthy(month)) {
		if toNumber(month) > 0 {
			M = toNumber(month) - 1
		} else {
			M = now.getMonth()
		}
	}
	h, m, s, ms := orZero(t.get("hours")), orZero(t.get("minutes")), orZero(t.get("seconds")), orZero(t.get("milliseconds"))
	if z := t.get("zone"); truthy(z) {
		off := toNumber(propOf(z, "offset"))
		return &Date{timeClip(dateUTC(y, M, d, h, m, s, toNumber(ms)+off*60*1000))}
	}
	if truthy(utc) {
		return &Date{timeClip(dateUTC(y, M, d, h, m, s, ms))}
	}
	newDate := NewDate(y, M, d, h, m, s, ms)
	if week := t.get("week"); truthy(week) {
		newDate = construct1(newDate).callDayjs("week", week).call("toDate").(*Date)
	}
	return newDate
}
