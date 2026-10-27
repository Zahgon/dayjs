package dayjs

import (
	"math"
	"regexp"
	"strings"
)

// The function-valued properties of the locales, ported from src/locale/*.js. Functions without
// free variables are shared by name (localePure); the others are registered under
// "<locale>.<property path>" (localeFuncs).

var (
	localePure  = map[string]Func{}
	localeFuncs = map[string]Func{}
)

func localeFn(key string) Func {
	fn, ok := localeFuncs[key]
	if !ok {
		panic("dayjs: no port of locale function " + key)
	}
	return fn
}

// toInt32 is ~~x.
func toInt32(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return float64(int32(uint32(int64(math.Mod(math.Trunc(x), 4294967296)))))
}

func str(v any) string { return toJSString(v) }

func num(v any) float64 { return toNumber(v) }

func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func meridiemBy(limits []float64, names []string) Func {
	return func(a ...any) any {
		h := num(arg(a, 0))
		for i, l := range limits {
			if h < l {
				return names[i]
			}
		}
		return names[len(names)-1]
	}
}

func digitMapper(re *regexp.Regexp, table map[string]string) func(string) string {
	return func(s string) string {
		return replaceAllRegexp(re, s, func(m string) string {
			if v, ok := table[m]; ok {
				return v
			}
			return "undefined"
		})
	}
}

func stringArg(v any, what string) string {
	s, ok := v.(string)
	if !ok {
		propOf(v, "replace")
		throw("TypeError", what+".replace is not a function")
	}
	return s
}

var (
	reDigit     = regexp.MustCompile(`\d`)
	reArabicNum = regexp.MustCompile(`[١٢٣٤٥٦٧٨٩٠]`)
	reBengali   = regexp.MustCompile(`[১২৩৪৫৬৭৮৯০]`)
	reComma     = regexp.MustCompile(`,`)
	reArComma   = regexp.MustCompile(`،`)

	monthsInFormat   = jsRegexp(`D[oD]?(\[[^[\]]*\]|\s)+MMMM?`)
	monthsInFormatLt = jsRegexp(`D[oD]?(\[[^\[\]]*\]|\s)+MMMM?|MMMM?(\[[^\[\]]*\]|\s)+D[oD]?`)
	monthsInFormatPl = regexp.MustCompile(`D MMMM`)
)

func arabicDigits(digits string) map[string]string {
	m := map[string]string{}
	for i, r := range []rune(digits) {
		m[string(rune('0'+i))] = string(r)
	}
	return m
}

func invert(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[v] = k
	}
	return out
}

// monthsFn is the months/monthsShort function of be, hr, lt, pl, ru and uk.
func monthsFn(re *regexp.Regexp, format, standalone string) Func {
	f, s := strings.Split(format, "_"), strings.Split(standalone, "_")
	return func(a ...any) any {
		inst, ok := arg(a, 0).(*Dayjs)
		if !ok {
			throw("TypeError", "dayjsInstance.month is not a function")
		}
		list := s
		if re.MatchString(str(arg(a, 1))) {
			list = f
		}
		return index(toAnyList(list), num(inst.call("month")))
	}
}

func toAnyList(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// slavicPlural is the plural(word, num) of be, ru and uk.
func slavicPlural(word any, n float64) any {
	forms := toAnyList(strings.Split(stringArg(word, "word"), "_"))
	m10, m100 := math.Mod(n, 10), math.Mod(n, 100)
	if m10 == 1 && m100 != 11 {
		return index(forms, 0)
	}
	if m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20) {
		return index(forms, 1)
	}
	return index(forms, 2)
}

// relativeWithPlural is relativeTimeWithPlural of be, ru and uk.
func relativeWithPlural(forms func(withoutSuffix bool) map[string]string, m, h [2]string) Func {
	return func(a ...any) any {
		number, withoutSuffix, key := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2))
		format := forms(withoutSuffix)
		if key == "m" && m[0] != "" {
			return pick(withoutSuffix, m[0], m[1])
		}
		if key == "h" && h[0] != "" {
			return pick(withoutSuffix, h[0], h[1])
		}
		var word any = Undefined
		if w, ok := format[key]; ok {
			word = w
		}
		return str(number) + " " + str(slavicPlural(word, toNumber(number)))
	}
}

func registerKeys(prefix string, keys string, fn Func) {
	for _, k := range strings.Fields(keys) {
		localeFuncs[prefix+k] = fn
	}
}

// csskPlural is the plural(n) of cs and sk.
func csskPlural(n float64) bool { return n > 1 && n < 5 && toInt32(n/10) != 1 }

type czechWords struct {
	s, sPast, m, mFut, mPast, mm, mmFew, mmPast, h, hFut, hPast, hh, hhFew, hhPast        string
	d, dPast, dd, ddFew, ddPast, M, MPast, MM, MMFew, MMPast, y, yPast, yy, yyFew, yyPast string
}

func czechTranslate(w czechWords) Func {
	return func(a ...any) any {
		number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
		result := str(number) + " "
		n := toNumber(number)
		either := withoutSuffix || isFuture
		few := func(f, other, past string) any {
			if either {
				return result + pick(csskPlural(n), f, other)
			}
			return result + past
		}
		switch key {
		case "s":
			return pick(either, w.s, w.sPast)
		case "m":
			if withoutSuffix {
				return w.m
			}
			return pick(isFuture, w.mFut, w.mPast)
		case "mm":
			return few(w.mmFew, w.mm, w.mmPast)
		case "h":
			if withoutSuffix {
				return w.h
			}
			return pick(isFuture, w.hFut, w.hPast)
		case "hh":
			return few(w.hhFew, w.hh, w.hhPast)
		case "d":
			return pick(either, w.d, w.dPast)
		case "dd":
			return few(w.ddFew, w.dd, w.ddPast)
		case "M":
			return pick(either, w.M, w.MPast)
		case "MM":
			return few(w.MMFew, w.MM, w.MMPast)
		case "y":
			return pick(either, w.y, w.yPast)
		case "yy":
			return few(w.yyFew, w.yy, w.yyPast)
		}
		return Undefined
	}
}

// germanTexts is the texts table of de, de-at and de-ch.
var germanTexts = map[string][]string{
	"s": {"ein paar Sekunden"}, "m": {"eine Minute", "einer Minute"}, "mm": {"%d Minuten"},
	"h": {"eine Stunde", "einer Stunde"}, "hh": {"%d Stunden"}, "d": {"ein Tag", "einem Tag"},
	"dd": {"%d Tage", "%d Tagen"}, "M": {"ein Monat", "einem Monat"}, "MM": {"%d Monate", "%d Monaten"},
	"y": {"ein Jahr", "einem Jahr"}, "yy": {"%d Jahre", "%d Jahren"},
}

func germanFormatter(a ...any) any {
	key := str(arg(a, 2))
	l, ok := germanTexts[key]
	if !ok {
		throw("TypeError", "Cannot read properties of undefined (reading 'replace')")
	}
	text := l[0]
	if len(l) == 2 && !truthy(arg(a, 1)) {
		text = l[1]
	}
	return replaceFirst(text, "%d", arg(a, 0))
}

func init() {
	localePure["ordIdentity"] = func(a ...any) any { return arg(a, 0) }
	localePure["ordAm"] = func(a ...any) any { return str(arg(a, 0)) + "ኛ" }
	localePure["meridiemAr"] = func(a ...any) any { return pick(num(arg(a, 0)) > 12, "م", "ص") }
	localePure["meridiemBe"] = meridiemBy([]float64{4, 12, 17}, []string{"ночы", "раніцы", "дня", "вечара"})
	localePure["ordBg"] = func(a ...any) any {
		n := arg(a, 0)
		last2 := math.Mod(num(n), 100)
		if last2 > 10 && last2 < 20 {
			return str(n) + "-ти"
		}
		switch math.Mod(num(n), 10) {
		case 1:
			return str(n) + "-ви"
		case 2:
			return str(n) + "-ри"
		case 7, 8:
			return str(n) + "-ми"
		}
		return str(n) + "-ти"
	}
	localePure["meridiemBnBd"] = meridiemBy([]float64{4, 6, 12, 15, 18, 20}, []string{"রাত", "ভোর", "সকাল", "দুপুর", "বিকাল", "সন্ধ্যা", "রাত"})
	suffixOrdinal := func(s []any) Func {
		return func(a ...any) any {
			n := arg(a, 0)
			v := math.Mod(num(n), 100)
			r := index(s, math.Mod(v-20, 10))
			if !truthy(r) {
				r = index(s, v)
			}
			if !truthy(r) {
				r = s[0]
			}
			return "[" + str(n) + str(r) + "]"
		}
	}
	localePure["ordBnBd"] = suffixOrdinal([]any{"ই", "লা", "রা", "ঠা", "শে"})
	localePure["ordEn"] = suffixOrdinal([]any{"th", "st", "nd", "rd"})
	localePure["meridiemBr"] = func(a ...any) any { return pick(num(arg(a, 0)) < 12, "a.m.", "g.m.") }
	localePure["ordCa"] = func(a ...any) any {
		n := arg(a, 0)
		var ord string
		switch {
		case strictEq(n, 1.0) || strictEq(n, 3.0):
			ord = "r"
		case strictEq(n, 2.0):
			ord = "n"
		case strictEq(n, 4.0):
			ord = "t"
		default:
			ord = "è"
		}
		return str(n) + ord
	}
	localePure["ordDot"] = func(a ...any) any { return str(arg(a, 0)) + "." }
	localePure["ordMasc"] = func(a ...any) any { return str(arg(a, 0)) + "º" }
	localePure["ordFr"] = func(a ...any) any {
		n := arg(a, 0)
		return str(n) + pick(strictEq(n, 1.0), "er", "")
	}
	hu := func(prefix bool, word, past string) Func {
		return func(a ...any) any {
			s := truthy(arg(a, 3)) || truthy(arg(a, 1))
			head := "egy"
			if prefix {
				head = str(arg(a, 0))
			}
			return head + " " + pick(s, word, past)
		}
	}
	localePure["huM"] = hu(false, "hónap", "hónapja")
	localePure["huMM"] = hu(true, "hónap", "hónapja")
	localePure["hud"] = hu(false, "nap", "napja")
	localePure["hudd"] = hu(true, "nap", "napja")
	localePure["huh"] = hu(false, "óra", "órája")
	localePure["huhh"] = hu(true, "óra", "órája")
	localePure["huy"] = hu(false, "év", "éve")
	localePure["huyy"] = hu(true, "év", "éve")
	perc := func(prefix bool, word string) Func {
		return func(a ...any) any {
			s := truthy(arg(a, 3)) || truthy(arg(a, 1))
			head := word
			if prefix {
				head = str(arg(a, 0)) + " " + word
			}
			return head + pick(s, "", "e")
		}
	}
	localePure["hum"] = perc(false, "egy perc")
	localePure["humm"] = perc(true, "perc")
	localePure["hus"] = perc(false, "néhány másodperc")
	localePure["meridiemJa"] = func(a ...any) any { return pick(num(arg(a, 0)) < 12, "午前", "午後") }
	localePure["ordJa"] = func(a ...any) any { return str(arg(a, 0)) + "日" }
	localePure["meridiemKm"] = func(a ...any) any { return pick(num(arg(a, 0)) > 12, "ល្ងាច", "ព្រឹក") }
	localePure["meridiemKo"] = func(a ...any) any { return pick(num(arg(a, 0)) < 12, "오전", "오후") }
	localePure["ordKo"] = func(a ...any) any { return str(arg(a, 0)) + "일" }
	localePure["meridiemKu"] = func(a ...any) any { return pick(num(arg(a, 0)) < 12, "پ.ن", "د.ن") }
	nepali := []rune("०१२३४५६७८९")
	localePure["ordNe"] = func(a ...any) any {
		return replaceAllRegexp(reDigit, str(arg(a, 0)), func(m string) string { return string(nepali[m[0]-'0']) })
	}
	localePure["ordNl"] = func(a ...any) any {
		n := arg(a, 0)
		ste := strictEq(n, 1.0) || strictEq(n, 8.0) || num(n) >= 20
		return "[" + str(n) + pick(ste, "ste", "de") + "]"
	}
	localePure["meridiemRu"] = meridiemBy([]float64{4, 12, 17}, []string{"ночи", "утра", "дня", "вечера"})
	localePure["ordSv"] = func(a ...any) any {
		n := arg(a, 0)
		b := math.Mod(num(n), 10)
		return "[" + str(n) + pick(b == 1 || b == 2, "a", "e") + "]"
	}
	localePure["meridiemZh"] = func(a ...any) any {
		hm := num(arg(a, 0))*100 + num(arg(a, 1))
		switch {
		case hm < 600:
			return "凌晨"
		case hm < 900:
			return "早上"
		case hm < 1100:
			return "上午"
		case hm < 1300:
			return "中午"
		case hm < 1800:
			return "下午"
		}
		return "晚上"
	}
	period := func(week string) Func {
		return func(a ...any) any {
			if strictEq(arg(a, 1), "W") {
				return str(arg(a, 0)) + week
			}
			return str(arg(a, 0)) + "日"
		}
	}
	localePure["ordZh"] = period("周")
	localePure["ordZhTw"] = period("週")

	// ar, bn, bn-bd and ku: digits and commas.
	arDigits := arabicDigits("٠١٢٣٤٥٦٧٨٩")
	toArabic, fromArabic := digitMapper(reDigit, arDigits), digitMapper(reArabicNum, invert(arDigits))
	localeFuncs["ar.postformat"] = func(a ...any) any {
		return reComma.ReplaceAllString(toArabic(stringArg(arg(a, 0), "string")), "،")
	}
	localeFuncs["ar.preparse"] = func(a ...any) any {
		return reArComma.ReplaceAllString(fromArabic(stringArg(arg(a, 0), "string")), ",")
	}
	localeFuncs["ku.postformat"] = localeFuncs["ar.postformat"]
	localeFuncs["ku.preparse"] = localeFuncs["ar.preparse"]
	bnDigits := arabicDigits("০১২৩৪৫৬৭৮৯")
	toBengali, fromBengali := digitMapper(reDigit, bnDigits), digitMapper(reBengali, invert(bnDigits))
	for _, l := range []string{"bn", "bn-bd"} {
		localeFuncs[l+".postformat"] = func(a ...any) any { return toBengali(stringArg(arg(a, 0), "string")) }
		localeFuncs[l+".preparse"] = func(a ...any) any { return fromBengali(stringArg(arg(a, 0), "string")) }
	}

	// Months functions.
	localeFuncs["be.months"] = monthsFn(monthsInFormat, "студзеня_лютага_сакавіка_красавіка_траўня_чэрвеня_ліпеня_жніўня_верасня_кастрычніка_лістапада_снежня", "студзень_лютый_сакавік_красавік_травень_чэрвень_ліпень_жнівень_верасень_кастрычнік_лістапад_снежань")
	localeFuncs["be.monthsShort"] = monthsFn(monthsInFormat, "студ_лют_сак_крас_трав_чэрв_ліп_жнів_вер_каст_ліст_снеж.", "студ_лют_сак_крас_трав_чэрв_ліп_жнів_вер_каст_ліст_снеж")
	localeFuncs["hr.months"] = monthsFn(monthsInFormat, "siječnja_veljače_ožujka_travnja_svibnja_lipnja_srpnja_kolovoza_rujna_listopada_studenoga_prosinca", "siječanj_veljača_ožujak_travanj_svibanj_lipanj_srpanj_kolovoz_rujan_listopad_studeni_prosinac")
	localeFuncs["lt.months"] = monthsFn(monthsInFormatLt, "sausio_vasario_kovo_balandžio_gegužės_birželio_liepos_rugpjūčio_rugsėjo_spalio_lapkričio_gruodžio", "sausis_vasaris_kovas_balandis_gegužė_birželis_liepa_rugpjūtis_rugsėjis_spalis_lapkritis_gruodis")
	localeFuncs["pl.months"] = monthsFn(monthsInFormatPl, "stycznia_lutego_marca_kwietnia_maja_czerwca_lipca_sierpnia_września_października_listopada_grudnia", "styczeń_luty_marzec_kwiecień_maj_czerwiec_lipiec_sierpień_wrzesień_październik_listopad_grudzień")
	localeFuncs["ru.months"] = monthsFn(monthsInFormat, "января_февраля_марта_апреля_мая_июня_июля_августа_сентября_октября_ноября_декабря", "январь_февраль_март_апрель_май_июнь_июль_август_сентябрь_октябрь_ноябрь_декабрь")
	localeFuncs["ru.monthsShort"] = monthsFn(monthsInFormat, "янв._февр._мар._апр._мая_июня_июля_авг._сент._окт._нояб._дек.", "янв._февр._март_апр._май_июнь_июль_авг._сент._окт._нояб._дек.")
	localeFuncs["uk.months"] = monthsFn(monthsInFormat, "січня_лютого_березня_квітня_травня_червня_липня_серпня_вересня_жовтня_листопада_грудня", "січень_лютий_березень_квітень_травень_червень_липень_серпень_вересень_жовтень_листопад_грудень")

	// be, ru, uk: relativeTimeWithPlural.
	registerKeys("be.relativeTime.", "MM dd h hh m mm yy", relativeWithPlural(func(ws bool) map[string]string {
		return map[string]string{
			"ss": pick(ws, "секунда_секунды_секунд", "секунду_секунды_секунд"),
			"mm": pick(ws, "хвіліна_хвіліны_хвілін", "хвіліну_хвіліны_хвілін"),
			"hh": pick(ws, "гадзіна_гадзіны_гадзін", "гадзіну_гадзіны_гадзін"),
			"dd": "дзень_дні_дзён", "MM": "месяц_месяцы_месяцаў", "yy": "год_гады_гадоў",
		}
	}, [2]string{"хвіліна", "хвіліну"}, [2]string{"гадзіна", "гадзіну"}))
	registerKeys("ru.relativeTime.", "MM dd hh m mm yy", relativeWithPlural(func(ws bool) map[string]string {
		return map[string]string{
			"mm": pick(ws, "минута_минуты_минут", "минуту_минуты_минут"),
			"hh": "час_часа_часов", "dd": "день_дня_дней", "MM": "месяц_месяца_месяцев", "yy": "год_года_лет",
		}
	}, [2]string{"минута", "минуту"}, [2]string{}))
	registerKeys("uk.relativeTime.", "MM dd h hh m mm yy", relativeWithPlural(func(ws bool) map[string]string {
		return map[string]string{
			"ss": pick(ws, "секунда_секунди_секунд", "секунду_секунди_секунд"),
			"mm": pick(ws, "хвилина_хвилини_хвилин", "хвилину_хвилини_хвилин"),
			"hh": pick(ws, "година_години_годин", "годину_години_годин"),
			"dd": "день_дні_днів", "MM": "місяць_місяці_місяців", "yy": "рік_роки_років",
		}
	}, [2]string{"хвилина", "хвилину"}, [2]string{"година", "годину"}))

	// br.
	var lastNumber func(n float64) float64
	lastNumber = func(n float64) float64 {
		if n > 9 {
			return lastNumber(math.Mod(n, 10))
		}
		return n
	}
	softMutation := func(text string) string {
		table := map[string]string{"m": "v", "b": "v", "d": "z"}
		first := jsSlice(text, 0.0, 1.0)
		head, ok := table[first]
		if !ok {
			head = "undefined"
		}
		return head + jsSlice(text, 1.0, Undefined)
	}
	registerKeys("br.relativeTime.", "MM dd mm", func(a ...any) any {
		number, key := arg(a, 0), str(arg(a, 2))
		format := map[string]string{"mm": "munutenn", "MM": "miz", "dd": "devezh"}
		text, ok := format[key]
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading 'charAt')")
		}
		if strictEq(number, 2.0) {
			text = softMutation(text)
		}
		return str(number) + " " + text
	})
	localeFuncs["br.relativeTime.yy"] = func(a ...any) any {
		number := arg(a, 0)
		switch lastNumber(num(number)) {
		case 1, 3, 4, 5, 9:
			return str(number) + " bloaz"
		}
		return str(number) + " vloaz"
	}

	// cs, sk.
	registerKeys("cs.relativeTime.", "M MM d dd h hh m mm s y yy", czechTranslate(czechWords{
		s: "pár sekund", sPast: "pár sekundami", m: "minuta", mFut: "minutu", mPast: "minutou",
		mm: "minut", mmFew: "minuty", mmPast: "minutami", h: "hodina", hFut: "hodinu", hPast: "hodinou",
		hh: "hodin", hhFew: "hodiny", hhPast: "hodinami", d: "den", dPast: "dnem", dd: "dní", ddFew: "dny",
		ddPast: "dny", M: "měsíc", MPast: "měsícem", MM: "měsíců", MMFew: "měsíce", MMPast: "měsíci",
		y: "rok", yPast: "rokem", yy: "let", yyFew: "roky", yyPast: "lety",
	}))
	registerKeys("sk.relativeTime.", "M MM d dd h hh m mm s y yy", czechTranslate(czechWords{
		s: "pár sekúnd", sPast: "pár sekundami", m: "minúta", mFut: "minútu", mPast: "minútou",
		mm: "minút", mmFew: "minúty", mmPast: "minútami", h: "hodina", hFut: "hodinu", hPast: "hodinou",
		hh: "hodín", hhFew: "hodiny", hhPast: "hodinami", d: "deň", dPast: "dňom", dd: "dní", ddFew: "dni",
		ddPast: "dňami", M: "mesiac", MPast: "mesiacom", MM: "mesiacov", MMFew: "mesiace", MMPast: "mesiacmi",
		y: "rok", yPast: "rokom", yy: "rokov", yyFew: "roky", yyPast: "rokmi",
	}))

	// de, de-at, de-ch.
	for _, l := range []string{"de", "de-at", "de-ch"} {
		registerKeys(l+".relativeTime.", "M MM d dd h hh m mm s y yy", germanFormatter)
	}

	// et.
	etFormat := map[string][]string{
		"s": {"mõne sekundi", "mõni sekund", "paar sekundit"}, "m": {"ühe minuti", "üks minut"},
		"mm": {"%d minuti", "%d minutit"}, "h": {"ühe tunni", "tund aega", "üks tund"},
		"hh": {"%d tunni", "%d tundi"}, "d": {"ühe päeva", "üks päev"}, "M": {"kuu aja", "kuu aega", "üks kuu"},
		"MM": {"%d kuu", "%d kuud"}, "y": {"ühe aasta", "aasta", "üks aasta"}, "yy": {"%d aasta", "%d aastat"},
	}
	registerKeys("et.relativeTime.", "M MM d h hh m mm s y yy", func(a ...any) any {
		number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
		f, ok := etFormat[key]
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading '2')")
		}
		var text string
		if withoutSuffix {
			text = f[1]
			if len(f) > 2 {
				text = f[2]
			}
		} else {
			text = pick(isFuture, f[0], f[1])
		}
		return replaceFirst(text, "%d", number)
	})

	// fi.
	fiPast := map[string]string{"s": "muutama sekunti", "m": "minuutti", "mm": "%d minuuttia", "h": "tunti",
		"hh": "%d tuntia", "d": "päivä", "dd": "%d päivää", "M": "kuukausi", "MM": "%d kuukautta", "y": "vuosi",
		"yy": "%d vuotta"}
	fiFuture := map[string]string{"s": "muutaman sekunnin", "m": "minuutin", "mm": "%d minuutin", "h": "tunnin",
		"hh": "%d tunnin", "d": "päivän", "dd": "%d päivän", "M": "kuukauden", "MM": "%d kuukauden", "y": "vuoden",
		"yy": "%d vuoden"}
	fiPastNumbers := toAnyList(strings.Split("nolla_yksi_kaksi_kolme_neljä_viisi_kuusi_seitsemän_kahdeksan_yhdeksän", "_"))
	fiFutureNumbers := toAnyList(strings.Split("nollan_yhden_kahden_kolmen_neljän_viiden_kuuden_seitsemän_kahdeksan_yhdeksän", "_"))
	registerKeys("fi.relativeTime.", "M MM d dd h hh m mm s y yy", func(a ...any) any {
		number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
		words, numbers := fiPast, fiPastNumbers
		if isFuture && !withoutSuffix {
			words, numbers = fiFuture, fiFutureNumbers
		}
		result, ok := words[key]
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading 'replace')")
		}
		if toNumber(number) < 10 {
			return replaceFirst(result, "%d", index(numbers, toNumber(number)))
		}
		return replaceFirst(result, "%d", number)
	})

	// he.
	heTexts := map[string]string{"s": "מספר שניות", "ss": "%d שניות", "m": "דקה", "mm": "%d דקות", "h": "שעה",
		"hh": "%d שעות", "hh2": "שעתיים", "d": "יום", "dd": "%d ימים", "dd2": "יומיים", "M": "חודש",
		"MM": "%d חודשים", "MM2": "חודשיים", "y": "שנה", "yy": "%d שנים", "yy2": "שנתיים"}
	registerKeys("he.relativeTime.", "M MM d dd h hh m mm s y yy", func(a ...any) any {
		number, key := arg(a, 0), str(arg(a, 2))
		k := key
		if strictEq(number, 2.0) {
			k += "2"
		}
		text, ok := heTexts[k]
		if !ok {
			text, ok = heTexts[key]
		}
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading 'replace')")
		}
		return replaceFirst(text, "%d", number)
	})

	// is.
	isTexts := map[string][]string{
		"s": {"nokkrar sekúndur", "nokkrar sekúndur", "nokkrum sekúndum"}, "m": {"mínúta", "mínútu", "mínútu"},
		"mm": {"mínútur", "mínútur", "mínútum"}, "h": {"klukkustund", "klukkustund", "klukkustund"},
		"hh": {"klukkustundir", "klukkustundir", "klukkustundum"}, "d": {"dagur", "dag", "degi"},
		"dd": {"dagar", "daga", "dögum"}, "M": {"mánuður", "mánuð", "mánuði"},
		"MM": {"mánuðir", "mánuði", "mánuðum"}, "y": {"ár", "ár", "ári"}, "yy": {"ár", "ár", "árum"},
	}
	registerKeys("is.relativeTime.", "M MM d dd h hh m mm s y yy", func(a ...any) any {
		number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
		suffixIndex := 2
		if isFuture {
			suffixIndex = 1
		}
		idx := suffixIndex
		if withoutSuffix {
			idx = 0
		}
		corrected := key
		if utf16Len(key) == 2 && math.Mod(toNumber(number), 10) == 1 {
			corrected = jsSlice(key, 0.0, 1.0)
		}
		unit, ok := isTexts[corrected]
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading '0')")
		}
		template := unit[idx]
		if utf16Len(key) != 1 {
			template = "%d " + template
		}
		return replaceFirst(template, "%d", number)
	})

	// pl.
	plPlural := func(n float64) bool {
		return math.Mod(n, 10) < 5 && math.Mod(n, 10) > 1 && math.Mod(toInt32(n/10), 10) != 1
	}
	registerKeys("pl.relativeTime.", "MM h hh m mm yy", func(a ...any) any {
		number, withoutSuffix, key := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2))
		result := str(number) + " "
		few := plPlural(toNumber(number))
		switch key {
		case "m":
			return pick(withoutSuffix, "minuta", "minutę")
		case "mm":
			return result + pick(few, "minuty", "minut")
		case "h":
			return pick(withoutSuffix, "godzina", "godzinę")
		case "hh":
			return result + pick(few, "godziny", "godzin")
		case "MM":
			return result + pick(few, "miesiące", "miesięcy")
		case "yy":
			return result + pick(few, "lata", "lat")
		}
		return Undefined
	})

	// sl.
	dual := func(n float64) bool { return math.Mod(n, 100) == 2 }
	threeFour := func(n float64) bool { return math.Mod(n, 100) == 3 || math.Mod(n, 100) == 4 }
	registerKeys("sl.relativeTime.", "M MM d dd h hh m mm s y yy", func(a ...any) any {
		number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
		result := str(number) + " "
		n := toNumber(number)
		either := withoutSuffix || isFuture
		forms := func(d2, d2p, tf, tfp, other, otherp string) any {
			if dual(n) {
				return result + pick(either, d2, d2p)
			}
			if tf != "" && threeFour(n) {
				return result + pick(either, tf, tfp)
			}
			return result + pick(either, other, otherp)
		}
		switch key {
		case "s":
			return pick(either, "nekaj sekund", "nekaj sekundami")
		case "m":
			return pick(withoutSuffix, "ena minuta", "eno minuto")
		case "mm":
			return forms("minuti", "minutama", "minute", "minutami", "minut", "minutami")
		case "h":
			return pick(withoutSuffix, "ena ura", "eno uro")
		case "hh":
			return forms("uri", "urama", "ure", "urami", "ur", "urami")
		case "d":
			return pick(either, "en dan", "enim dnem")
		case "dd":
			return forms("dneva", "dnevoma", "", "", "dni", "dnevi")
		case "M":
			return pick(either, "en mesec", "enim mesecem")
		case "MM":
			return forms("meseca", "mesecema", "mesece", "meseci", "mesecev", "meseci")
		case "y":
			return pick(either, "eno leto", "enim letom")
		case "yy":
			return forms("leti", "letoma", "leta", "leti", "let", "leti")
		}
		return Undefined
	})

	// sr, sr-cyrl.
	serbian := func(words map[string][]string, oneYear, yearForm, years string) Func {
		return func(a ...any) any {
			number, withoutSuffix, key, isFuture := arg(a, 0), truthy(arg(a, 1)), str(arg(a, 2)), truthy(arg(a, 3))
			wordKey, ok := words[key]
			if !ok {
				throw("TypeError", "Cannot read properties of undefined (reading '0')")
			}
			if utf16Len(key) == 1 {
				if key == "y" && withoutSuffix {
					return oneYear
				}
				return pick(isFuture || withoutSuffix, wordKey[0], wordKey[1])
			}
			n := toNumber(number)
			m10, m100 := math.Mod(n, 10), math.Mod(n, 100)
			var word string
			if m10 >= 1 && m10 <= 4 && (m100 < 10 || m100 >= 20) {
				word = pick(m10 == 1, wordKey[0], wordKey[1])
			} else {
				word = wordKey[2]
			}
			if key == "yy" && withoutSuffix && word == yearForm {
				return str(number) + " " + years
			}
			return replaceFirst(word, "%d", number)
		}
	}
	registerKeys("sr.relativeTime.", "M MM d dd h hh m mm y yy", serbian(map[string][]string{
		"m": {"jedan minut", "jednog minuta"}, "mm": {"%d minut", "%d minuta", "%d minuta"},
		"h": {"jedan sat", "jednog sata"}, "hh": {"%d sat", "%d sata", "%d sati"},
		"d": {"jedan dan", "jednog dana"}, "dd": {"%d dan", "%d dana", "%d dana"},
		"M": {"jedan mesec", "jednog meseca"}, "MM": {"%d mesec", "%d meseca", "%d meseci"},
		"y": {"jednu godinu", "jedne godine"}, "yy": {"%d godinu", "%d godine", "%d godina"},
	}, "jedna godina", "%d godinu", "godina"))
	registerKeys("sr-cyrl.relativeTime.", "M MM d dd h hh m mm y yy", serbian(map[string][]string{
		"m": {"један минут", "једног минута"}, "mm": {"%d минут", "%d минута", "%d минута"},
		"h": {"један сат", "једног сата"}, "hh": {"%d сат", "%d сата", "%d сати"},
		"d": {"један дан", "једног дана"}, "dd": {"%d дан", "%d дана", "%d дана"},
		"M": {"један месец", "једног месеца"}, "MM": {"%d месец", "%d месеца", "%d месеци"},
		"y": {"једну годину", "једне године"}, "yy": {"%d годину", "%d године", "%d година"},
	}, "једна година", "%d годину", "година"))
}
