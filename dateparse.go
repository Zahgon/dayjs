package dayjs

// Date.parse as V8 implements it (src/date/dateparser*.{h,cc}): an ES5 ISO 8601 pass, then the
// legacy parser that accepts what Safari accepts, both over one tokenizer.

import (
	"math"
	"unicode/utf16"
)

const dpNone = math.MaxInt32

// smiMax is V8's Smi::kMaxValue in Node.js, which is built without pointer compression.
const smiMax = 1<<31 - 1

type dpKeyword int

const (
	kwInvalid dpKeyword = iota
	kwMonthName
	kwTimeZoneName
	kwTimeSeparator
	kwAmPm
)

type dpTag int

const (
	tagInvalid    dpTag = -6
	tagUnknown    dpTag = -5
	tagWhiteSpace dpTag = -4
	tagNumber     dpTag = -3
	tagSymbol     dpTag = -2
	tagEnd        dpTag = -1
)

type dpToken struct {
	tag    dpTag // a keyword token's tag is its dpKeyword (>= 0)
	length int
	value  int
}

func (t dpToken) isInvalid() bool                { return t.tag == tagInvalid }
func (t dpToken) isNumber() bool                 { return t.tag == tagNumber }
func (t dpToken) isSymbolChar(c byte) bool       { return t.tag == tagSymbol && t.value == int(c) }
func (t dpToken) isWhiteSpace() bool             { return t.tag == tagWhiteSpace }
func (t dpToken) isEnd() bool                    { return t.tag == tagEnd }
func (t dpToken) isKeyword() bool                { return t.tag >= 0 }
func (t dpToken) keyword() dpKeyword             { return dpKeyword(t.tag) }
func (t dpToken) isKeywordType(k dpKeyword) bool { return t.tag == dpTag(k) }
func (t dpToken) isFixedLengthNumber(n int) bool { return t.isNumber() && t.length == n }
func (t dpToken) isAsciiSign() bool {
	return t.tag == tagSymbol && (t.value == '-' || t.value == '+')
}
func (t dpToken) asciiSign() int { return 44 - t.value }
func (t dpToken) isKeywordZ() bool {
	return t.tag == dpTag(kwTimeZoneName) && t.length == 1 && t.value == 0
}

var dpKeywords = []struct {
	prefix [3]uint16
	kind   dpKeyword
	value  int
}{
	{[3]uint16{'j', 'a', 'n'}, kwMonthName, 1},
	{[3]uint16{'f', 'e', 'b'}, kwMonthName, 2},
	{[3]uint16{'m', 'a', 'r'}, kwMonthName, 3},
	{[3]uint16{'a', 'p', 'r'}, kwMonthName, 4},
	{[3]uint16{'m', 'a', 'y'}, kwMonthName, 5},
	{[3]uint16{'j', 'u', 'n'}, kwMonthName, 6},
	{[3]uint16{'j', 'u', 'l'}, kwMonthName, 7},
	{[3]uint16{'a', 'u', 'g'}, kwMonthName, 8},
	{[3]uint16{'s', 'e', 'p'}, kwMonthName, 9},
	{[3]uint16{'o', 'c', 't'}, kwMonthName, 10},
	{[3]uint16{'n', 'o', 'v'}, kwMonthName, 11},
	{[3]uint16{'d', 'e', 'c'}, kwMonthName, 12},
	{[3]uint16{'a', 'm', 0}, kwAmPm, 0},
	{[3]uint16{'p', 'm', 0}, kwAmPm, 12},
	{[3]uint16{'u', 't', 0}, kwTimeZoneName, 0},
	{[3]uint16{'u', 't', 'c'}, kwTimeZoneName, 0},
	{[3]uint16{'z', 0, 0}, kwTimeZoneName, 0},
	{[3]uint16{'g', 'm', 't'}, kwTimeZoneName, 0},
	{[3]uint16{'c', 'd', 't'}, kwTimeZoneName, -5},
	{[3]uint16{'c', 's', 't'}, kwTimeZoneName, -6},
	{[3]uint16{'e', 'd', 't'}, kwTimeZoneName, -4},
	{[3]uint16{'e', 's', 't'}, kwTimeZoneName, -5},
	{[3]uint16{'m', 'd', 't'}, kwTimeZoneName, -6},
	{[3]uint16{'m', 's', 't'}, kwTimeZoneName, -7},
	{[3]uint16{'p', 'd', 't'}, kwTimeZoneName, -7},
	{[3]uint16{'p', 's', 't'}, kwTimeZoneName, -8},
	{[3]uint16{'t', 0, 0}, kwTimeSeparator, 0},
}

func dpLookup(prefix [3]uint16, length int) (dpKeyword, int) {
	for _, k := range dpKeywords {
		if k.prefix == prefix && (length <= 3 || k.kind == kwMonthName) {
			return k.kind, k.value
		}
	}
	return kwInvalid, 0
}

// isJSWhiteSpaceUnit is ECMAScript's WhiteSpace for one UTF-16 unit.
func isJSWhiteSpaceUnit(c uint16) bool {
	switch c {
	case 0x09, 0x0B, 0x0C, 0x20, 0xA0, 0xFEFF, 0x1680, 0x202F, 0x205F, 0x3000:
		return true
	}
	return c >= 0x2000 && c <= 0x200A
}

func isJSLineTerminatorUnit(c uint16) bool {
	return c == 0x0A || c == 0x0D || c == 0x2028 || c == 0x2029
}

type dpReader struct {
	buf   []uint16
	index int
	ch    uint16
}

func (r *dpReader) next() {
	if r.index < len(r.buf) {
		r.ch = r.buf[r.index]
	} else {
		r.ch = 0
	}
	r.index++
}

func (r *dpReader) skip(c uint16) bool {
	if r.ch == c {
		r.next()
		return true
	}
	return false
}

func (r *dpReader) isEnd() bool          { return r.ch == 0 }
func (r *dpReader) isDigit() bool        { return r.ch >= '0' && r.ch <= '9' }
func (r *dpReader) isAlphaOrAbove() bool { return r.ch >= 'A' }
func (r *dpReader) isWhiteSpace() bool   { return isJSWhiteSpaceUnit(r.ch) }

func (r *dpReader) readUnsignedNumeral() int {
	n, i := 0, 0
	for r.ch == '0' {
		r.next()
	}
	for r.isDigit() {
		if i < 9 {
			n = n*10 + int(r.ch-'0')
		}
		i++
		r.next()
	}
	return n
}

func (r *dpReader) readWord() ([3]uint16, int) {
	var prefix [3]uint16
	length := 0
	for ; r.isAlphaOrAbove() && !r.isWhiteSpace(); r.next() {
		if length < 3 {
			prefix[length] = r.ch | 0x20
		}
		length++
	}
	return prefix, length
}

func (r *dpReader) skipWhiteSpace() bool {
	if isJSWhiteSpaceUnit(r.ch) || isJSLineTerminatorUnit(r.ch) {
		r.next()
		return true
	}
	return false
}

func (r *dpReader) skipParentheses() bool {
	if r.ch != '(' {
		return false
	}
	balance := 0
	for {
		if r.ch == ')' {
			balance--
		} else if r.ch == '(' {
			balance++
		}
		r.next()
		if !(balance > 0 && r.ch != 0) {
			break
		}
	}
	return true
}

type dpScanner struct {
	in   *dpReader
	peek dpToken
}

func (s *dpScanner) scan() dpToken {
	in := s.in
	pre := in.index
	if in.isEnd() {
		return dpToken{tag: tagEnd, value: -1}
	}
	if in.isDigit() {
		n := in.readUnsignedNumeral()
		return dpToken{tag: tagNumber, length: in.index - pre, value: n}
	}
	for _, c := range []uint16{':', '-', '+', '.', ')'} {
		if in.skip(c) {
			return dpToken{tag: tagSymbol, length: 1, value: int(c)}
		}
	}
	if in.isAlphaOrAbove() && !in.isWhiteSpace() {
		prefix, length := in.readWord()
		kind, value := dpLookup(prefix, length)
		return dpToken{tag: dpTag(kind), length: length, value: value}
	}
	if in.skipWhiteSpace() {
		return dpToken{tag: tagWhiteSpace, length: in.index - pre}
	}
	if in.skipParentheses() {
		return dpToken{tag: tagUnknown, length: 1, value: -1}
	}
	in.next()
	return dpToken{tag: tagUnknown, length: 1, value: -1}
}

func (s *dpScanner) next() dpToken {
	t := s.peek
	s.peek = s.scan()
	return t
}

func (s *dpScanner) skipSymbol(c byte) bool {
	if s.peek.isSymbolChar(c) {
		s.peek = s.scan()
		return true
	}
	return false
}

func between(x, lo, hi int) bool { return x >= lo && x <= hi }

type dayComposer struct {
	comp       [3]int
	index      int
	namedMonth int
	isISODate  bool
}

func (d *dayComposer) add(n int) bool {
	if d.index == 3 {
		return false
	}
	d.comp[d.index] = n
	d.index++
	return true
}

func (d *dayComposer) write(out *[8]float64) bool {
	if d.index < 1 {
		return false
	}
	for d.index < 3 {
		d.comp[d.index] = 1
		d.index++
	}
	year, month, day := 0, dpNone, dpNone
	if d.namedMonth == dpNone {
		if d.isISODate || (d.index == 3 && !between(d.comp[0], 1, 31)) {
			year, month, day = d.comp[0], d.comp[1], d.comp[2]
		} else {
			month, day = d.comp[0], d.comp[1]
			if d.index == 3 {
				year = d.comp[2]
			}
		}
	} else {
		month = d.namedMonth
		if d.index == 1 {
			day = d.comp[0]
		} else if !between(d.comp[0], 1, 31) {
			year, day = d.comp[0], d.comp[1]
		} else {
			day, year = d.comp[0], d.comp[1]
		}
	}
	if !d.isISODate {
		if between(year, 0, 49) {
			year += 2000
		} else if between(year, 50, 99) {
			year += 1900
		}
	}
	if year < -smiMax-1 || year > smiMax || !between(month, 1, 12) || !between(day, 1, 31) {
		return false
	}
	out[0], out[1], out[2] = float64(year), float64(month-1), float64(day)
	return true
}

type timeComposer struct {
	comp       [4]int
	index      int
	hourOffset int
}

func (t *timeComposer) isEmpty() bool { return t.index == 0 }
func (t *timeComposer) isExpecting(n int) bool {
	return (t.index == 1 && between(n, 0, 59)) || (t.index == 2 && between(n, 0, 59)) ||
		(t.index == 3 && between(n, 0, 999))
}
func (t *timeComposer) add(n int) bool {
	if t.index < 4 {
		t.comp[t.index] = n
		t.index++
		return true
	}
	return false
}
func (t *timeComposer) addFinal(n int) bool {
	if !t.add(n) {
		return false
	}
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	return true
}

func (t *timeComposer) write(out *[8]float64) bool {
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	hour, minute, second, ms := t.comp[0], t.comp[1], t.comp[2], t.comp[3]
	if t.hourOffset != dpNone {
		if !between(hour, 0, 12) {
			return false
		}
		hour %= 12
		hour += t.hourOffset
	}
	if !between(hour, 0, 23) || !between(minute, 0, 59) || !between(second, 0, 59) || !between(ms, 0, 999) {
		if hour != 24 || minute != 0 || second != 0 || ms != 0 {
			return false
		}
	}
	out[3], out[4], out[5], out[6] = float64(hour), float64(minute), float64(second), float64(ms)
	return true
}

type tzComposer struct {
	sign, hour, minute int
}

func (z *tzComposer) set(offsetHours int) {
	if offsetHours < 0 {
		z.sign = -1
	} else {
		z.sign = 1
	}
	z.hour = offsetHours * z.sign
	z.minute = 0
}
func (z *tzComposer) setSign(sign int) {
	if sign < 0 {
		z.sign = -1
	} else {
		z.sign = 1
	}
}
func (z *tzComposer) isExpecting(n int) bool {
	return z.hour != dpNone && z.minute == dpNone && between(n, 0, 59)
}
func (z *tzComposer) isUTC() bool   { return z.hour == 0 && z.minute == 0 }
func (z *tzComposer) isEmpty() bool { return z.hour == dpNone }

func (z *tzComposer) write(out *[8]float64) bool {
	if z.sign != dpNone {
		if z.hour == dpNone {
			z.hour = 0
		}
		if z.minute == dpNone {
			z.minute = 0
		}
		total := uint64(uint32(z.hour))*3600 + uint64(uint32(z.minute))*60
		total &= 0xFFFFFFFF
		if total > smiMax {
			return false
		}
		secs := int(total)
		if z.sign < 0 {
			secs = -secs
		}
		out[7] = float64(secs)
	} else {
		out[7] = math.NaN()
	}
	return true
}

// readMilliseconds takes the first three significant digits of a numeral as milliseconds.
func readMilliseconds(t dpToken) int {
	number, length := t.value, t.length
	if length < 3 {
		if length == 1 {
			number *= 100
		} else if length == 2 {
			number *= 10
		}
	} else if length > 3 {
		if length > 9 {
			length = 9
		}
		factor := 1
		for {
			factor *= 10
			length--
			if length <= 3 {
				break
			}
		}
		number /= factor
	}
	return number
}

func parseES5DateTime(s *dpScanner, day *dayComposer, tm *timeComposer, tz *tzComposer) dpToken {
	if s.peek.isAsciiSign() {
		signToken := s.next()
		if !s.peek.isFixedLengthNumber(6) {
			return signToken
		}
		sign := signToken.asciiSign()
		year := s.next().value
		if sign < 0 && year == 0 {
			return signToken
		}
		day.add(sign * year)
	} else if s.peek.isFixedLengthNumber(4) {
		day.add(s.next().value)
	} else {
		return s.next()
	}
	if s.skipSymbol('-') {
		if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 1, 12) {
			return s.next()
		}
		day.add(s.next().value)
		if s.skipSymbol('-') {
			if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 1, 31) {
				return s.next()
			}
			day.add(s.next().value)
		}
	}
	if !s.peek.isKeywordType(kwTimeSeparator) {
		if !s.peek.isEnd() {
			return s.next()
		}
	} else {
		s.next()
		if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 0, 24) {
			return dpToken{tag: tagInvalid}
		}
		hourIs24 := s.peek.value == 24
		tm.add(s.next().value)
		if !s.skipSymbol(':') {
			return dpToken{tag: tagInvalid}
		}
		if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 0, 59) || (hourIs24 && s.peek.value > 0) {
			return dpToken{tag: tagInvalid}
		}
		tm.add(s.next().value)
		if s.skipSymbol(':') {
			if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 0, 59) || (hourIs24 && s.peek.value > 0) {
				return dpToken{tag: tagInvalid}
			}
			tm.add(s.next().value)
			if s.skipSymbol('.') {
				if !s.peek.isNumber() || (hourIs24 && s.peek.value > 0) {
					return dpToken{tag: tagInvalid}
				}
				tm.add(readMilliseconds(s.next()))
			}
		}
		if s.peek.isKeywordZ() {
			s.next()
			tz.set(0)
		} else if s.peek.isSymbolChar('+') || s.peek.isSymbolChar('-') {
			if s.next().value == '+' {
				tz.setSign(1)
			} else {
				tz.setSign(-1)
			}
			if s.peek.isFixedLengthNumber(4) {
				hourmin := s.next().value
				hour, min := hourmin/100, hourmin%100
				if !between(hour, 0, 23) || !between(min, 0, 59) {
					return dpToken{tag: tagInvalid}
				}
				tz.hour, tz.minute = hour, min
			} else {
				if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 0, 23) {
					return dpToken{tag: tagInvalid}
				}
				tz.hour = s.next().value
				if !s.skipSymbol(':') {
					return dpToken{tag: tagInvalid}
				}
				if !s.peek.isFixedLengthNumber(2) || !between(s.peek.value, 0, 59) {
					return dpToken{tag: tagInvalid}
				}
				tz.minute = s.next().value
			}
		}
		if !s.peek.isEnd() {
			return dpToken{tag: tagInvalid}
		}
	}
	if tz.isEmpty() && tm.isEmpty() {
		tz.set(0)
	}
	day.isISODate = true
	return dpToken{tag: tagEnd, value: -1}
}

// parseDateFields runs V8's DateParser::Parse; out holds year, month (0-based), day, hour,
// minute, second, millisecond and the UTC offset in seconds (NaN for local time).
func parseDateFields(str string) (out [8]float64, ok bool) {
	in := &dpReader{buf: utf16.Encode([]rune(str))}
	in.next()
	s := &dpScanner{in: in}
	s.peek = s.scan()
	day := &dayComposer{namedMonth: dpNone}
	tm := &timeComposer{hourOffset: dpNone}
	tz := &tzComposer{sign: dpNone, hour: dpNone, minute: dpNone}

	next := parseES5DateTime(s, day, tm, tz)
	if next.isInvalid() {
		return out, false
	}
	hasReadNumber := day.index > 0
	for token := next; !token.isEnd(); token = s.next() {
		switch {
		case token.isNumber():
			hasReadNumber = true
			n := token.value
			if s.skipSymbol(':') {
				if s.skipSymbol(':') {
					if !tm.isEmpty() {
						return out, false
					}
					tm.add(n)
					tm.add(0)
				} else {
					if !tm.add(n) {
						return out, false
					}
					if s.peek.isSymbolChar('.') {
						s.next()
					}
				}
			} else if s.skipSymbol('.') && tm.isExpecting(n) {
				tm.add(n)
				if !s.peek.isNumber() {
					return out, false
				}
				ms := readMilliseconds(s.next())
				if ms < 0 {
					return out, false
				}
				tm.addFinal(ms)
			} else if tz.isExpecting(n) {
				tz.minute = n
			} else if tm.isExpecting(n) {
				tm.addFinal(n)
				p := s.peek
				if !p.isEnd() && !p.isWhiteSpace() && !p.isKeywordZ() && !p.isAsciiSign() {
					return out, false
				}
			} else {
				if !day.add(n) {
					return out, false
				}
				s.skipSymbol('-')
			}
		case token.isKeyword():
			switch {
			case token.keyword() == kwAmPm && !tm.isEmpty():
				tm.hourOffset = token.value
			case token.keyword() == kwMonthName:
				day.namedMonth = token.value
				s.skipSymbol('-')
			case token.keyword() == kwTimeZoneName && hasReadNumber:
				tz.set(token.value)
			default:
				if hasReadNumber {
					return out, false
				}
				if s.peek.isNumber() {
					return out, false
				}
			}
		case token.isAsciiSign() && (tz.isUTC() || !tm.isEmpty()):
			tz.setSign(token.asciiSign())
			n, length := 0, 0
			if s.peek.isNumber() {
				nt := s.next()
				length, n = nt.length, nt.value
			}
			hasReadNumber = true
			if s.peek.isSymbolChar(':') {
				tz.hour = n
				tz.minute = dpNone
			} else if length == 2 || length == 1 {
				tz.hour, tz.minute = n, 0
			} else if length == 4 || length == 3 {
				tz.hour, tz.minute = n/100, n%100
			} else {
				return out, false
			}
		case (token.isAsciiSign() || token.isSymbolChar(')')) && hasReadNumber:
			return out, false
		}
	}
	ok = day.write(&out) && tm.write(&out) && tz.write(&out)
	return out, ok
}

// dateParse is Date.parse(str) (and new Date(str)).
func dateParse(str string) float64 {
	out, ok := parseDateFields(str)
	if !ok {
		return math.NaN()
	}
	date := makeDate(makeDay(out[0], out[1], out[2]), makeTime(out[3], out[4], out[5], out[6]))
	if math.IsNaN(out[7]) {
		if math.IsNaN(date) || date < -maxTimeBeforeUTCMs || date > maxTimeBeforeUTCMs {
			return math.NaN()
		}
		date = localToUTC(date)
	} else {
		date -= out[7] * 1000
		if math.IsNaN(date) || date < -maxTimeMs || date > maxTimeMs {
			return math.NaN()
		}
	}
	return timeClip(date)
}
