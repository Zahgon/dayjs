package dayjs

import (
	"math"
	"regexp"
	"strings"
	"unicode/utf16"
)

// jsWhiteSpaceClass is the contents of a character class matching what \s matches in a
// JavaScript regular expression (WhiteSpace and LineTerminator), which is wider than Go's \s.
const jsWhiteSpaceClass = `\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`

// jsRegexp compiles a JavaScript regular expression source (no flags other than i, given as a
// (?i) prefix) as an equivalent RE2 expression: \s, \S and . take their JavaScript meanings.
func jsRegexp(src string) *regexp.Regexp {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '\\' && i+1 < len(src):
			n := src[i+1]
			i++
			switch {
			case n == 's' && inClass:
				b.WriteString(jsWhiteSpaceClass)
			case n == 's':
				b.WriteString("[" + jsWhiteSpaceClass + "]")
			case n == 'S' && !inClass:
				b.WriteString("[^" + jsWhiteSpaceClass + "]")
			default:
				b.WriteByte('\\')
				b.WriteByte(n)
			}
		case c == '[' && !inClass:
			inClass = true
			b.WriteByte(c)
			if i+1 < len(src) && src[i+1] == '^' {
				b.WriteByte('^')
				i++
			}
			if i+1 < len(src) && src[i+1] == '[' {
				b.WriteString(`\[`)
				i++
			}
		case c == ']' && inClass:
			inClass = false
			b.WriteByte(c)
		case c == '[' && inClass:
			b.WriteString(`\[`)
		case c == '.' && !inClass:
			b.WriteString(`[^\n\r\x{2028}\x{2029}]`)
		default:
			b.WriteByte(c)
		}
	}
	return regexp.MustCompile(b.String())
}

// replaceAllSubmatchFunc is String.prototype.replace with a global regular expression and a
// replacement function receiving the match and its groups (has tells which groups took part).
func replaceAllSubmatchFunc(re *regexp.Regexp, s string, fn func(m []string, has []bool) string) string {
	locs := re.FindAllStringSubmatchIndex(s, -1)
	if locs == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, loc := range locs {
		b.WriteString(s[last:loc[0]])
		n := len(loc) / 2
		m := make([]string, n)
		has := make([]bool, n)
		for i := 0; i < n; i++ {
			if loc[2*i] >= 0 {
				m[i] = s[loc[2*i]:loc[2*i+1]]
				has[i] = true
			}
		}
		b.WriteString(fn(m, has))
		last = loc[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// replaceAllRegexp is s.replace(/re/g, fn) with a function of the match only.
func replaceAllRegexp(re *regexp.Regexp, s string, fn func(string) string) string {
	return replaceAllSubmatchFunc(re, s, func(m []string, _ []bool) string { return fn(m[0]) })
}

// replaceFirst is s.replace(pattern, replacement) with a string pattern: the first occurrence
// is replaced and the replacement's $ patterns are expanded.
func replaceFirst(s, pattern string, replacement any) string {
	i := strings.Index(s, pattern)
	if i < 0 {
		return s
	}
	rep := toJSString(replacement)
	return s[:i] + substitution(rep, pattern, s[:i], s[i+len(pattern):]) + s[i+len(pattern):]
}

// substitution is GetSubstitution for a match without capture groups.
func substitution(rep, matched, before, after string) string {
	if !strings.Contains(rep, "$") {
		return rep
	}
	var b strings.Builder
	for i := 0; i < len(rep); i++ {
		if rep[i] != '$' || i+1 >= len(rep) {
			b.WriteByte(rep[i])
			continue
		}
		switch rep[i+1] {
		case '$':
			b.WriteByte('$')
			i++
		case '&':
			b.WriteString(matched)
			i++
		case '`':
			b.WriteString(before)
			i++
		case '\'':
			b.WriteString(after)
			i++
		default:
			b.WriteByte('$')
		}
	}
	return b.String()
}

// units16 and from16 convert between Go strings and JavaScript's UTF-16 code units.
func units16(s string) []uint16 { return utf16.Encode([]rune(s)) }

func from16(u []uint16) string { return string(utf16.Decode(u)) }

// relIndex resolves a slice index the way String.prototype.slice does.
func relIndex(v any, n int, def int) int {
	if isUndef(v) {
		return def
	}
	f := toIntegerOrInfinity(toNumber(v))
	if f < 0 {
		f += float64(n)
		if f < 0 {
			f = 0
		}
	}
	if f > float64(n) {
		f = float64(n)
	}
	return int(f)
}

// jsSlice is String.prototype.slice.
func jsSlice(s string, start, end any) string {
	if isASCII(s) {
		a, b := relIndex(start, len(s), 0), relIndex(end, len(s), len(s))
		if a >= b {
			return ""
		}
		return s[a:b]
	}
	u := units16(s)
	a, b := relIndex(start, len(u), 0), relIndex(end, len(u), len(u))
	if a >= b {
		return ""
	}
	return from16(u[a:b])
}

// jsSliceAny is v.slice(start, end) for a value that must be a string (or an array).
func jsSliceAny(v any, start, end any) any {
	switch x := v.(type) {
	case string:
		return jsSlice(x, start, end)
	case []any:
		a, b := relIndex(start, len(x), 0), relIndex(end, len(x), len(x))
		if a >= b {
			return []any{}
		}
		return append([]any(nil), x[a:b]...)
	}
	propOf(v, "slice")
	throw("TypeError", "slice is not a function")
	return nil
}

// jsSubstring is String.prototype.substring with numeric bounds on an ASCII string.
func jsSubstring(s string, start, end int) string {
	if end > len(s) {
		end = len(s)
	}
	if start > end {
		start, end = end, start
	}
	return s[start:end]
}

// jsAdd is the + operator on primitive values.
func jsAdd(a, b any) any {
	_, as := a.(string)
	_, bs := b.(string)
	if as || bs {
		return toJSString(a) + toJSString(b)
	}
	return toNumber(a) + toNumber(b)
}

// strictEq is === on the values the port passes around.
func strictEq(a, b any) bool {
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case undefinedType:
		return isUndef(b)
	case nil:
		return b == nil
	}
	return a == b
}

// looseEq is == between any value and a string.
func looseEqString(a any, s string) bool {
	switch x := a.(type) {
	case string:
		return x == s
	case float64:
		return x == stringToNumber(s)
	case bool:
		n := 0.0
		if x {
			n = 1
		}
		return n == stringToNumber(s)
	case undefinedType, nil:
		return false
	case *Date:
		return x.toString() == s
	}
	return toJSString(a) == s
}

// isPlainObject reports whether obj.constructor.name is "Object": an object literal.
func isPlainObject(v any) bool {
	switch v.(type) {
	case Obj, *Object:
		return true
	case nil, undefinedType:
		throw("TypeError", "Cannot read properties of "+toJSString(v)+" (reading 'constructor')")
	}
	return false
}

// objectKeys is Object.keys on an object literal.
func objectKeys(v any) []string {
	switch o := v.(type) {
	case *Object:
		return o.Keys()
	case Obj:
		return o.Keys()
	case []any:
		ks := make([]string, len(o))
		for i := range o {
			ks[i] = numberToString(float64(i))
		}
		return ks
	case string:
		ks := make([]string, utf16Len(o))
		for i := range ks {
			ks[i] = numberToString(float64(i))
		}
		return ks
	case nil, undefinedType:
		throw("TypeError", "Cannot convert undefined or null to object")
	}
	return nil
}

// arrayIndexOf is Array.prototype.indexOf with strict equality.
func arrayIndexOf(arr []any, v any) int {
	for i, x := range arr {
		if strictEq(x, v) {
			return i
		}
	}
	return -1
}

// toArray returns the elements of an array value.
func toArray(v any) ([]any, bool) {
	switch a := v.(type) {
	case []any:
		return a, true
	case []string:
		out := make([]any, len(a))
		for i, s := range a {
			out[i] = s
		}
		return out, true
	}
	return nil, false
}

// mapArray is arr.map(fn), throwing when the value is not an array.
func mapArray(v any, fn func(any) any) []any {
	arr, ok := toArray(v)
	if !ok {
		propOf(v, "map")
		throw("TypeError", "map is not a function")
	}
	out := make([]any, len(arr))
	for i, x := range arr {
		out[i] = fn(x)
	}
	return out
}

// mathCeil and mathFloor keep JavaScript's signed zeros.
func mathCeil(x float64) float64 { return math.Ceil(x) }

func mathFloor(x float64) float64 { return math.Floor(x) }
