package dayjs

// This file holds the pieces of JavaScript semantics the port leans on: number-to-string and
// string-to-number conversions, Math.round, parseInt and parseFloat, case mapping, trimming, and
// the ordered object literal (Obj) used for every object argument and result.

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Obj is a JavaScript object literal written as alternating keys and values, in key order:
// Obj{"years", 1, "months", 2}. Key order is significant: Day.js iterates objects in insertion
// order. A missing key is undefined; a key whose value is nil is null.
type Obj []any

// undefinedType is the type of Undefined.
type undefinedType struct{}

// Undefined is the JavaScript undefined value, for an Obj entry that is present but undefined.
var Undefined = undefinedType{}

// Error is a JavaScript exception. Every exception the original throws is raised as a panic
// carrying a *Error whose Name is the JavaScript error class (TypeError, RangeError, ...) and
// whose Message is the exception message.
type Error struct {
	Name    string
	Message string
}

func (e *Error) Error() string { return e.Name + ": " + e.Message }

func throw(name, msg string) { panic(&Error{Name: name, Message: msg}) }

// ---------------------------------------------------------------------------------------------
// ordered objects

// Object is a mutable insertion-ordered object used internally.
type Object struct {
	keys []string
	vals map[string]any
}

func newObj() *Object { return &Object{vals: map[string]any{}} }

// objFrom copies an Obj; a later duplicate key overwrites the value but keeps the first position,
// as in an object literal.
func objFrom(o Obj) *Object {
	r := newObj()
	for i := 0; i+1 < len(o); i += 2 {
		k, _ := o[i].(string)
		r.set(k, o[i+1])
	}
	return r
}

func (o *Object) has(k string) bool { _, ok := o.vals[k]; return ok }

func (o *Object) get(k string) any {
	v, ok := o.vals[k]
	if !ok {
		return Undefined
	}
	return v
}

func (o *Object) set(k string, v any) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *Object) del(k string) {
	if _, ok := o.vals[k]; !ok {
		return
	}
	delete(o.vals, k)
	for i, x := range o.keys {
		if x == k {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
}

func (o *Object) clone() *Object {
	r := &Object{keys: append([]string(nil), o.keys...), vals: make(map[string]any, len(o.vals))}
	for k, v := range o.vals {
		r.vals[k] = v
	}
	return r
}

// assign is Object spread: {...o, ...src}.
func (o *Object) assign(src *Object) *Object {
	if src == nil {
		return o
	}
	for _, k := range src.keys {
		o.set(k, src.vals[k])
	}
	return o
}

func (o *Object) toObj() Obj {
	r := make(Obj, 0, 2*len(o.keys))
	for _, k := range o.keys {
		r = append(r, k, o.vals[k])
	}
	return r
}

// num reads a numeric property; ok is false when it is undefined.
func (o *Object) num(k string) (float64, bool) {
	v, ok := o.vals[k]
	if !ok || isUndef(v) {
		return math.NaN(), false
	}
	return toNumber(v), true
}

// numOr reads a property as a number, NaN when undefined (what arithmetic on undefined gives).
func (o *Object) numOr(k string) float64 {
	f, _ := o.num(k)
	return f
}

// Get returns the value stored under key and whether it is present.
func (o Obj) Get(key string) (any, bool) {
	var v any
	found := false
	for i := 0; i+1 < len(o); i += 2 {
		if k, _ := o[i].(string); k == key {
			v, found = o[i+1], true
		}
	}
	return v, found
}

// Keys returns the keys in order, without duplicates.
func (o Obj) Keys() []string { return objFrom(o).keys }

// opt reads a property of an optional object argument; a nil Obj reads as {}.
func opt(o Obj, key string) any {
	if v, ok := o.Get(key); ok {
		return v
	}
	return Undefined
}

// ---------------------------------------------------------------------------------------------
// type tests and conversions

func isUndef(v any) bool { _, ok := v.(undefinedType); return ok }

// isNullish is v == null in JavaScript (null or undefined).
func isNullish(v any) bool { return v == nil || isUndef(v) }

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	}
	return 0, false
}

// isNumber is typeof v === "number".
func isNumber(v any) bool { _, ok := asFloat(v); return ok }

func isString(v any) bool { _, ok := v.(string); return ok }

// typeOf is the typeof operator.
func typeOf(v any) string {
	switch v.(type) {
	case undefinedType:
		return "undefined"
	case nil:
		return "object"
	case string:
		return "string"
	case bool:
		return "boolean"
	}
	if isNumber(v) {
		return "number"
	}
	switch v.(type) {
	case Func, *Function:
		return "function"
	case *big.Int:
		return "bigint"
	}
	return "object"
}

// truthy is ToBoolean.
func truthy(v any) bool {
	switch x := v.(type) {
	case undefinedType, nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	}
	if f, ok := asFloat(v); ok {
		return f != 0 && !math.IsNaN(f)
	}
	return true
}

// toNumber is ToNumber for the primitive values the port sees.
func toNumber(v any) float64 {
	switch x := v.(type) {
	case undefinedType:
		return math.NaN()
	case nil:
		return 0
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		return stringToNumber(x)
	case *Dayjs:
		return x.ValueOf()
	case *Duration:
		return x.ValueOf()
	case *Date:
		return x.t
	case *big.Int:
		f, _ := new(big.Float).SetInt(x).Float64()
		return f
	case []any:
		// Number([]) is 0, Number([x]) is Number(String(x)), anything longer is NaN.
		switch len(x) {
		case 0:
			return 0
		case 1:
			return stringToNumber(toJSString(x[0]))
		}
		return math.NaN()
	}
	if f, ok := asFloat(v); ok {
		return f
	}
	return math.NaN()
}

// toJSString is ToString (and template literal interpolation).
func toJSString(v any) string {
	switch x := v.(type) {
	case undefinedType:
		return "undefined"
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return x
	case *Dayjs:
		return x.ToString()
	case *Duration:
		return "[object Object]"
	case *Date:
		return x.toString()
	case *big.Int:
		return x.String()
	case Func, *Function:
		return "function () { [native code] }"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			if !isNullish(e) {
				parts[i] = toJSString(e)
			}
		}
		return strings.Join(parts, ",")
	case Obj, *Object:
		return "[object Object]"
	}
	if f, ok := asFloat(v); ok {
		return numberToString(f)
	}
	return "[object Object]"
}

// numberToString is Number.prototype.toString() in radix 10.
func numberToString(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case x == 0:
		return "0"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x < 0:
		return "-" + numberToString(-x)
	}
	e := strconv.FormatFloat(x, 'e', -1, 64) // d.ddddde±XX, shortest round trip
	mant, exp, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	k := len(digits)
	ex, _ := strconv.Atoi(exp)
	n := ex + 1
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	sign := "+"
	if n-1 < 0 {
		sign = "-"
	}
	ae := n - 1
	if ae < 0 {
		ae = -ae
	}
	if k == 1 {
		return digits + "e" + sign + strconv.Itoa(ae)
	}
	return digits[:1] + "." + digits[1:] + "e" + sign + strconv.Itoa(ae)
}

// toFixed is Number.prototype.toFixed(f) for 0 <= f <= 100.
func toFixed(x float64, f int) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.Abs(x) >= 1e21 || math.IsInf(x, 0) {
		return numberToString(x)
	}
	sign := ""
	if x < 0 {
		sign = "-"
		x = -x
	}
	r := new(big.Rat).SetFloat64(x)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(f)), nil)
	r.Mul(r, new(big.Rat).SetInt(scale))
	r.Add(r, big.NewRat(1, 2))
	n := new(big.Int).Quo(r.Num(), r.Denom()) // floor for a positive rational
	s := n.String()
	if f > 0 {
		if len(s) <= f {
			s = strings.Repeat("0", f-len(s)+1) + s
		}
		s = s[:len(s)-f] + "." + s[len(s)-f:]
	}
	if sign == "-" && strings.Trim(s, "0.") == "" {
		// (-0.0000001).toFixed(2) is "-0.00": the sign survives.
		return "-" + s
	}
	return sign + s
}

// isJSWhitespace is WhiteSpace or LineTerminator, the set String.prototype.trim and \s use.
func isJSWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

func jsTrim(s string) string { return strings.TrimFunc(s, isJSWhitespace) }

func jsTrimStart(s string) string { return strings.TrimLeftFunc(s, isJSWhitespace) }

var (
	decimalLiteral = func(s string) bool {
		// StrUnsignedDecimalLiteral: digits [. digits] [exp] | . digits [exp]
		i, n := 0, len(s)
		d1 := 0
		for i < n && s[i] >= '0' && s[i] <= '9' {
			i++
			d1++
		}
		d2 := 0
		if i < n && s[i] == '.' {
			i++
			for i < n && s[i] >= '0' && s[i] <= '9' {
				i++
				d2++
			}
		}
		if d1 == 0 && d2 == 0 {
			return false
		}
		if i < n && (s[i] == 'e' || s[i] == 'E') {
			i++
			if i < n && (s[i] == '+' || s[i] == '-') {
				i++
			}
			d3 := 0
			for i < n && s[i] >= '0' && s[i] <= '9' {
				i++
				d3++
			}
			if d3 == 0 {
				return false
			}
		}
		return i == n
	}
)

// stringToNumber is ToNumber applied to a string.
func stringToNumber(s string) float64 {
	t := jsTrim(s)
	if t == "" {
		return 0
	}
	if len(t) > 2 && t[0] == '0' {
		base := 0
		switch t[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			v := 0.0
			for _, c := range t[2:] {
				d := digitVal(c)
				if d < 0 || d >= base {
					return math.NaN()
				}
				v = float64(v*float64(base)) + float64(d)
			}
			return v
		}
	}
	body := t
	sign := 1.0
	if body[0] == '+' || body[0] == '-' {
		if body[0] == '-' {
			sign = -1
		}
		body = body[1:]
	}
	if body == "Infinity" {
		return sign * math.Inf(1)
	}
	if !decimalLiteral(body) {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(body, 64) // overflow gives ±Inf, as in JavaScript
	return sign * f
}

func digitVal(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

// parseInt10 is parseInt(s, 10).
func parseInt10(s string) float64 {
	t := jsTrimStart(s)
	sign := 1.0
	if t != "" && (t[0] == '+' || t[0] == '-') {
		if t[0] == '-' {
			sign = -1
		}
		t = t[1:]
	}
	end := 0
	for end < len(t) && t[end] >= '0' && t[end] <= '9' {
		end++
	}
	if end == 0 {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(t[:end], 64)
	return sign * f
}

// parseFloatJS is parseFloat(s).
func parseFloatJS(s string) float64 {
	t := jsTrimStart(s)
	sign := 1.0
	body := t
	if body != "" && (body[0] == '+' || body[0] == '-') {
		if body[0] == '-' {
			sign = -1
		}
		body = body[1:]
	}
	if strings.HasPrefix(body, "Infinity") {
		return sign * math.Inf(1)
	}
	// longest prefix that is a StrUnsignedDecimalLiteral
	best := -1
	for i := 1; i <= len(body); i++ {
		if decimalLiteral(body[:i]) {
			best = i
		}
	}
	if best < 0 {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(body[:best], 64)
	return sign * f
}

// mathRound is Math.round.
func mathRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) || x == 0 {
		return x
	}
	if x > 0 && x < 0.5 {
		return 0
	}
	if x < 0 && x >= -0.5 {
		return math.Copysign(0, -1)
	}
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r
}

// mathSign is Math.sign.
func mathSign(x float64) float64 {
	switch {
	case math.IsNaN(x):
		return x
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return x
}

// isNegZero is Object.is(x, -0).
func isNegZero(x float64) bool { return x == 0 && math.Signbit(x) }

// isInteger is Number.isInteger.
func isIntegerF(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) && x == math.Trunc(x) }

// jsMod is the % operator on numbers.
func jsMod(a, b float64) float64 { return math.Mod(a, b) }

// ---------------------------------------------------------------------------------------------
// case mapping

// jsToLower is String.prototype.toLowerCase (full mappings, final sigma).
func jsToLower(s string) string {
	if isASCII(s) {
		return strings.ToLower(s)
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		switch r {
		case 0x130: // İ lowercases to i + combining dot above
			b.WriteString("i̇")
			continue
		case 0x3A3: // Σ: final sigma when preceded by a cased letter and not followed by one
			if finalSigma(runes, i) {
				b.WriteRune(0x3C2)
			} else {
				b.WriteRune(0x3C3)
			}
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func finalSigma(runes []rune, i int) bool {
	before := false
	for j := i - 1; j >= 0; j-- {
		if isCaseIgnorable(runes[j]) {
			continue
		}
		before = isCased(runes[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(runes); j++ {
		if isCaseIgnorable(runes[j]) {
			continue
		}
		return !isCased(runes[j])
	}
	return true
}

func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) || unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}

func isCaseIgnorable(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk) || r == '\'' || r == '.' || r == ':' || r == 0xAD || r == 0x2019
}

// upperSpecial lists characters whose full uppercase mapping is longer than one code unit.
var upperSpecial = map[rune]string{
	0xDF: "SS", 0x149: "ʼN", 0x1F0: "J̌", 0x390: "Ϊ́", 0x3B0: "Ϋ́",
	0x587: "ԵՒ", 0x1E96: "H̱", 0x1E97: "T̈", 0x1E98: "W̊", 0x1E99: "Y̊",
	0x1E9A: "Aʾ", 0xFB00: "FF", 0xFB01: "FI", 0xFB02: "FL", 0xFB03: "FFI", 0xFB04: "FFL", 0xFB05: "ST",
	0xFB06: "ST", 0xFB13: "ՄՆ", 0xFB14: "ՄԵ", 0xFB15: "ՄԻ", 0xFB16: "ՎՆ",
	0xFB17: "ՄԽ",
}

// jsToUpper is String.prototype.toUpperCase for one character.
func jsToUpperRune(r rune) string {
	if s, ok := upperSpecial[r]; ok {
		return s
	}
	if r >= 0x1F50 && r <= 0x1FFC {
		// Greek extended: characters with iota subscripts or breathing marks that uppercase to
		// several code points. unicode.ToUpper gives the single-code-point simple mapping, which
		// is correct for the rest of this block.
		switch r {
		case 0x1F50, 0x1F52, 0x1F54, 0x1F56, 0x1FB2, 0x1FB4, 0x1FB6, 0x1FB7, 0x1FC2, 0x1FC4, 0x1FC6,
			0x1FC7, 0x1FD2, 0x1FD3, 0x1FD6, 0x1FD7, 0x1FE2, 0x1FE3, 0x1FE4, 0x1FE6, 0x1FE7, 0x1FF2,
			0x1FF4, 0x1FF6, 0x1FF7:
			return "\x00\x00" // more than one code point; only the length matters to callers
		}
	}
	if r >= 0x1F80 && r <= 0x1FAF || r == 0x1FB3 || r == 0x1FBC || r == 0x1FC3 || r == 0x1FCC || r == 0x1FF3 || r == 0x1FFC {
		return "\x00\x00"
	}
	return string(unicode.ToUpper(r))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// utf16Len is String.prototype.length.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// sliceLast is s.slice(-n) on a string of BMP characters.
func sliceLast(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

// padStartStr is String.prototype.padStart(n, "0").
func padStartStr(s string, n int) string {
	if l := utf16Len(s); l < n {
		return strings.Repeat("0", n-l) + s
	}
	return s
}

// ---------------------------------------------------------------------------------------------
// exported object and function values

// NewObject builds an ordered object from an object literal.
func NewObject(o Obj) *Object { return objFrom(o) }

// Get returns the property key, Undefined when it is missing.
func (o *Object) Get(key string) any { return o.get(key) }

// Set sets the property key, appending it to the key order when it is new.
func (o *Object) Set(key string, v any) { o.set(key, v) }

// Has reports whether the object has the property key.
func (o *Object) Has(key string) bool { return o.has(key) }

// Keys returns the property names in insertion order.
func (o *Object) Keys() []string { return append([]string(nil), o.keys...) }

// Func is a JavaScript function value: it receives the call's arguments as JavaScript values
// (float64, string, bool, nil for null, Undefined, *Dayjs, ...) and returns one.
type Func func(args ...any) any

// Function is a JavaScript function value that also carries own properties, such as a locale's
// months function with its standalone (s) and format (f) name lists.
type Function struct {
	Call  Func
	Props *Object
}

// NewFunction returns a function value with the given own properties.
func NewFunction(call Func, props Obj) *Function { return &Function{Call: call, Props: objFrom(props)} }

// callable returns the Go function behind a JavaScript function value.
func callable(v any) (Func, bool) {
	switch f := v.(type) {
	case Func:
		return f, f != nil
	case func(args ...any) any:
		return f, f != nil
	case *Function:
		return f.Call, f != nil && f.Call != nil
	}
	return nil, false
}

// call invokes a JavaScript function value, throwing the TypeError JavaScript throws when it is
// not a function.
func call(v any, what string, args ...any) any {
	f, ok := callable(v)
	if !ok {
		throw("TypeError", what+" is not a function")
	}
	return f(args...)
}

// propOf is property access v[key] on the values the port passes around.
func propOf(v any, key string) any {
	switch x := v.(type) {
	case *Object:
		return x.get(key)
	case Obj:
		if r, ok := x.Get(key); ok {
			return r
		}
		return Undefined
	case *Function:
		if x.Props != nil {
			return x.Props.get(key)
		}
		return Undefined
	case []any:
		if key == "length" {
			return float64(len(x))
		}
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && i < len(x) && strconv.Itoa(i) == key {
			return x[i]
		}
		return Undefined
	case []string:
		if key == "length" {
			return float64(len(x))
		}
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && i < len(x) && strconv.Itoa(i) == key {
			return x[i]
		}
		return Undefined
	case string:
		if key == "length" {
			return float64(utf16Len(x))
		}
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && strconv.Itoa(i) == key {
			if u := units16(x); i < len(u) {
				return from16(u[i : i+1])
			}
		}
		return Undefined
	case undefinedType, nil:
		throw("TypeError", "Cannot read properties of "+toJSString(v)+" (reading '"+key+"')")
	}
	return Undefined
}

// index is v[i] for a numeric index.
func index(v any, i float64) any {
	if math.IsNaN(i) {
		return propOf(v, "NaN")
	}
	return propOf(v, numberToString(i))
}
