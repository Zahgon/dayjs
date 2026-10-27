package dayjs

// New is dayjs(...args): it passes the same arguments, in the same order, as the JavaScript
// call.
func New(args ...any) *Dayjs { return construct(args) }

// Locale is dayjs.locale(preset, object, isLocal). It returns the locale name, or false.
func Locale(args ...any) any { return staticCall("locale", args...) }

// IsDayjs is dayjs.isDayjs(value).
func IsDayjs(v any) bool { return isDayjs(v) }

// Unix is dayjs.unix(seconds).
func Unix(seconds any) *Dayjs { return staticDayjs("unix", seconds) }

// Call calls a JavaScript function value, such as a locale's ordinal or meridiem function.
func Call(fn any, args ...any) any { return call(fn, "fn", args...) }

// Utc is dayjs.utc(...args) (utc plugin).
func Utc(args ...any) *Dayjs { return staticDayjs("utc", args...) }

// Tz is dayjs.tz(...args) (timezone plugin).
func Tz(args ...any) *Dayjs {
	r := staticCall("tz", args...)
	if d, ok := r.(*Dayjs); ok {
		return d
	}
	throw("TypeError", "dayjs.tz(...) is not a Day.js object")
	return nil
}

func tzStatic(name string) any {
	f, ok := statics["tz"].(*Function)
	if !ok {
		throw("TypeError", "Cannot read properties of undefined (reading '"+name+"')")
	}
	return f.Props.get(name)
}

// TzGuess is dayjs.tz.guess() (timezone plugin).
func TzGuess() string { return toJSString(call(tzStatic("guess"), "dayjs.tz.guess")) }

// TzSetDefault is dayjs.tz.setDefault(timezone) (timezone plugin).
func TzSetDefault(timezone ...any) { call(tzStatic("setDefault"), "dayjs.tz.setDefault", timezone...) }

// NewDuration is dayjs.duration(input, unit) (duration plugin).
func NewDuration(args ...any) *Duration {
	r := staticCall("duration", args...)
	if d, ok := r.(*Duration); ok {
		return d
	}
	throw("TypeError", "dayjs.duration(...) is not a Duration")
	return nil
}

// IsDuration is dayjs.isDuration(value) (duration plugin).
func IsDuration(v any) bool { return truthy(staticCall("isDuration", v)) }

func dayjsOrNil(r any) *Dayjs {
	if d, ok := r.(*Dayjs); ok {
		return d
	}
	return nil
}

// Max is dayjs.max(...dates) (minMax plugin); nil where Day.js returns null.
func Max(dates ...any) *Dayjs { return dayjsOrNil(staticCall("max", dates...)) }

// Min is dayjs.min(...dates) (minMax plugin); nil where Day.js returns null.
func Min(dates ...any) *Dayjs { return dayjsOrNil(staticCall("min", dates...)) }

// IsMoment is dayjs.isMoment(value) (isMoment plugin).
func IsMoment(v any) bool { return truthy(staticCall("isMoment", v)) }

// GlobalLocaleData is dayjs.localeData() (localeData plugin).
func GlobalLocaleData() *LocaleData { return staticCall("localeData").(*LocaleData) }

// Months is dayjs.months() (localeData plugin).
func Months() any { return staticCall("months") }

// MonthsShort is dayjs.monthsShort() (localeData plugin).
func MonthsShort() any { return staticCall("monthsShort") }

// Weekdays is dayjs.weekdays(localeOrder) (localeData plugin).
func Weekdays(localeOrder ...any) any { return staticCall("weekdays", localeOrder...) }

// WeekdaysShort is dayjs.weekdaysShort(localeOrder) (localeData plugin).
func WeekdaysShort(localeOrder ...any) any { return staticCall("weekdaysShort", localeOrder...) }

// WeekdaysMin is dayjs.weekdaysMin(localeOrder) (localeData plugin).
func WeekdaysMin(localeOrder ...any) any { return staticCall("weekdaysMin", localeOrder...) }

// UpdateLocale is dayjs.updateLocale(locale, config) (updateLocale plugin); nil where Day.js
// returns undefined.
func UpdateLocale(args ...any) *Object {
	if o, ok := staticCall("updateLocale", args...).(*Object); ok {
		return o
	}
	return nil
}

// ---------------------------------------------------------------------------------------------
// Dayjs methods

func (d *Dayjs) num(name string, args ...any) float64 { return toNumber(d.call(name, args...)) }

func (d *Dayjs) boolean(name string, args ...any) bool { return truthy(d.call(name, args...)) }

func (d *Dayjs) str(name string, args ...any) string { return toJSString(d.call(name, args...)) }

func (d *Dayjs) setter(name string, v any) *Dayjs { return d.callDayjs(name, v) }

// Clone is clone().
func (d *Dayjs) Clone() *Dayjs { return d.callDayjs("clone") }

// IsValid is isValid().
func (d *Dayjs) IsValid() bool { return d.boolean("isValid") }

// Year is year().
func (d *Dayjs) Year() float64 { return d.num("year") }

// SetYear is year(value).
func (d *Dayjs) SetYear(v any) *Dayjs { return d.setter("year", v) }

// Month is month().
func (d *Dayjs) Month() float64 { return d.num("month") }

// SetMonth is month(value).
func (d *Dayjs) SetMonth(v any) *Dayjs { return d.setter("month", v) }

// Date is date().
func (d *Dayjs) Date() float64 { return d.num("date") }

// SetDate is date(value).
func (d *Dayjs) SetDate(v any) *Dayjs { return d.setter("date", v) }

// Day is day().
func (d *Dayjs) Day() float64 { return d.num("day") }

// SetDay is day(value).
func (d *Dayjs) SetDay(v any) *Dayjs { return d.setter("day", v) }

// Hour is hour().
func (d *Dayjs) Hour() float64 { return d.num("hour") }

// SetHour is hour(value).
func (d *Dayjs) SetHour(v any) *Dayjs { return d.setter("hour", v) }

// Minute is minute().
func (d *Dayjs) Minute() float64 { return d.num("minute") }

// SetMinute is minute(value).
func (d *Dayjs) SetMinute(v any) *Dayjs { return d.setter("minute", v) }

// Second is second().
func (d *Dayjs) Second() float64 { return d.num("second") }

// SetSecond is second(value).
func (d *Dayjs) SetSecond(v any) *Dayjs { return d.setter("second", v) }

// Millisecond is millisecond().
func (d *Dayjs) Millisecond() float64 { return d.num("millisecond") }

// SetMillisecond is millisecond(value).
func (d *Dayjs) SetMillisecond(v any) *Dayjs { return d.setter("millisecond", v) }

// Get is get(unit).
func (d *Dayjs) Get(unit any) any { return d.call("get", unit) }

// Set is set(unit, value).
func (d *Dayjs) Set(args ...any) *Dayjs { return d.callDayjs("set", args...) }

// Add is add(value, unit).
func (d *Dayjs) Add(args ...any) *Dayjs { return d.callDayjs("add", args...) }

// Subtract is subtract(value, unit).
func (d *Dayjs) Subtract(args ...any) *Dayjs { return d.callDayjs("subtract", args...) }

// StartOf is startOf(unit).
func (d *Dayjs) StartOf(args ...any) *Dayjs { return d.callDayjs("startOf", args...) }

// EndOf is endOf(unit).
func (d *Dayjs) EndOf(args ...any) *Dayjs { return d.callDayjs("endOf", args...) }

// Format is format(template).
func (d *Dayjs) Format(args ...any) string { return d.str("format", args...) }

// Diff is diff(date, unit, float).
func (d *Dayjs) Diff(args ...any) float64 { return d.num("diff", args...) }

// ValueOf is valueOf().
func (d *Dayjs) ValueOf() float64 { return d.num("valueOf") }

// Unix is unix().
func (d *Dayjs) Unix() float64 { return d.num("unix") }

// DaysInMonth is daysInMonth().
func (d *Dayjs) DaysInMonth() float64 { return d.num("daysInMonth") }

// ToDate is toDate().
func (d *Dayjs) ToDate() *Date { return d.call("toDate").(*Date) }

// ToJSON is toJSON(): the ISO string, or nil for an invalid date.
func (d *Dayjs) ToJSON() any { return d.call("toJSON") }

// ToISOString is toISOString().
func (d *Dayjs) ToISOString() string { return d.str("toISOString") }

// ToString is toString().
func (d *Dayjs) ToString() string { return d.str("toString") }

// UtcOffset is utcOffset().
func (d *Dayjs) UtcOffset() float64 { return d.num("utcOffset") }

// SetUtcOffset is utcOffset(offset, keepLocalTime) (utc plugin).
func (d *Dayjs) SetUtcOffset(args ...any) *Dayjs { return d.callDayjs("utcOffset", args...) }

// IsBefore is isBefore(date, unit).
func (d *Dayjs) IsBefore(args ...any) bool { return d.boolean("isBefore", args...) }

// IsAfter is isAfter(date, unit).
func (d *Dayjs) IsAfter(args ...any) bool { return d.boolean("isAfter", args...) }

// IsSame is isSame(date, unit).
func (d *Dayjs) IsSame(args ...any) bool { return d.boolean("isSame", args...) }

// Locale is locale(): the instance's locale name.
func (d *Dayjs) Locale() string { return d.str("locale") }

// SetLocale is locale(preset, object).
func (d *Dayjs) SetLocale(args ...any) *Dayjs { return d.callDayjs("locale", args...) }

// Utc is utc(keepLocalTime) (utc plugin).
func (d *Dayjs) Utc(args ...any) *Dayjs { return d.callDayjs("utc", args...) }

// Local is local() (utc plugin).
func (d *Dayjs) Local() *Dayjs { return d.callDayjs("local") }

// IsUTC is isUTC() (utc plugin).
func (d *Dayjs) IsUTC() bool { return d.boolean("isUTC") }

// Tz is tz(timezone, keepLocalTime) (timezone plugin).
func (d *Dayjs) Tz(args ...any) *Dayjs { return d.callDayjs("tz", args...) }

// OffsetName is offsetName(type) (timezone plugin).
func (d *Dayjs) OffsetName(args ...any) string { return d.str("offsetName", args...) }

// From is from(date, withoutSuffix) (relativeTime plugin).
func (d *Dayjs) From(args ...any) string { return d.str("from", args...) }

// To is to(date, withoutSuffix) (relativeTime plugin).
func (d *Dayjs) To(args ...any) string { return d.str("to", args...) }

// FromNow is fromNow(withoutSuffix) (relativeTime plugin).
func (d *Dayjs) FromNow(args ...any) string { return d.str("fromNow", args...) }

// ToNow is toNow(withoutSuffix) (relativeTime plugin).
func (d *Dayjs) ToNow(args ...any) string { return d.str("toNow", args...) }

// Calendar is calendar(referenceTime, formats) (calendar plugin).
func (d *Dayjs) Calendar(args ...any) string { return d.str("calendar", args...) }

// IsBetween is isBetween(a, b, unit, inclusivity) (isBetween plugin).
func (d *Dayjs) IsBetween(args ...any) bool { return d.boolean("isBetween", args...) }

// IsLeapYear is isLeapYear() (isLeapYear plugin).
func (d *Dayjs) IsLeapYear() bool { return d.boolean("isLeapYear") }

// IsSameOrAfter is isSameOrAfter(date, unit) (isSameOrAfter plugin).
func (d *Dayjs) IsSameOrAfter(args ...any) bool { return d.boolean("isSameOrAfter", args...) }

// IsSameOrBefore is isSameOrBefore(date, unit) (isSameOrBefore plugin).
func (d *Dayjs) IsSameOrBefore(args ...any) bool { return d.boolean("isSameOrBefore", args...) }

// IsToday is isToday() (isToday plugin).
func (d *Dayjs) IsToday() bool { return d.boolean("isToday") }

// IsTomorrow is isTomorrow() (isTomorrow plugin).
func (d *Dayjs) IsTomorrow() bool { return d.boolean("isTomorrow") }

// IsYesterday is isYesterday() (isYesterday plugin).
func (d *Dayjs) IsYesterday() bool { return d.boolean("isYesterday") }

// IsoWeekYear is isoWeekYear() (isoWeek plugin).
func (d *Dayjs) IsoWeekYear() float64 { return d.num("isoWeekYear") }

// IsoWeek is isoWeek() (isoWeek plugin).
func (d *Dayjs) IsoWeek() float64 { return d.num("isoWeek") }

// SetIsoWeek is isoWeek(week) (isoWeek plugin).
func (d *Dayjs) SetIsoWeek(v any) *Dayjs { return d.setter("isoWeek", v) }

// IsoWeekday is isoWeekday() (isoWeek plugin).
func (d *Dayjs) IsoWeekday() float64 { return d.num("isoWeekday") }

// SetIsoWeekday is isoWeekday(day) (isoWeek plugin).
func (d *Dayjs) SetIsoWeekday(v any) *Dayjs { return d.setter("isoWeekday", v) }

// IsoWeeksInYear is isoWeeksInYear() (isoWeeksInYear plugin).
func (d *Dayjs) IsoWeeksInYear() float64 { return d.num("isoWeeksInYear") }

// DayOfYear is dayOfYear() (dayOfYear plugin).
func (d *Dayjs) DayOfYear() float64 { return d.num("dayOfYear") }

// SetDayOfYear is dayOfYear(value) (dayOfYear plugin).
func (d *Dayjs) SetDayOfYear(v any) *Dayjs { return d.setter("dayOfYear", v) }

// Week is week() (weekOfYear plugin).
func (d *Dayjs) Week() float64 { return d.num("week") }

// SetWeek is week(value) (weekOfYear plugin).
func (d *Dayjs) SetWeek(v any) *Dayjs { return d.setter("week", v) }

// Weeks is weeks() (weekOfYear and pluralGetSet plugins).
func (d *Dayjs) Weeks() float64 { return d.num("weeks") }

// SetWeeks is weeks(value).
func (d *Dayjs) SetWeeks(v any) *Dayjs { return d.setter("weeks", v) }

// WeekYear is weekYear() (weekYear plugin).
func (d *Dayjs) WeekYear() float64 { return d.num("weekYear") }

// Weekday is weekday() (weekday plugin).
func (d *Dayjs) Weekday() float64 { return d.num("weekday") }

// SetWeekday is weekday(value) (weekday plugin).
func (d *Dayjs) SetWeekday(v any) *Dayjs { return d.setter("weekday", v) }

// Quarter is quarter() (quarterOfYear plugin).
func (d *Dayjs) Quarter() float64 { return d.num("quarter") }

// SetQuarter is quarter(value) (quarterOfYear plugin).
func (d *Dayjs) SetQuarter(v any) *Dayjs { return d.setter("quarter", v) }

// ToArray is toArray() (toArray plugin).
func (d *Dayjs) ToArray() []any { return d.call("toArray").([]any) }

// ToObject is toObject() (toObject plugin).
func (d *Dayjs) ToObject() *Object { return d.call("toObject").(*Object) }

// LocaleData is localeData() (localeData plugin).
func (d *Dayjs) LocaleData() *LocaleData { return d.call("localeData").(*LocaleData) }

// Milliseconds is milliseconds() (pluralGetSet plugin).
func (d *Dayjs) Milliseconds() float64 { return d.num("milliseconds") }

// SetMilliseconds is milliseconds(value) (pluralGetSet plugin).
func (d *Dayjs) SetMilliseconds(v any) *Dayjs { return d.setter("milliseconds", v) }

// Seconds is seconds() (pluralGetSet plugin).
func (d *Dayjs) Seconds() float64 { return d.num("seconds") }

// SetSeconds is seconds(value) (pluralGetSet plugin).
func (d *Dayjs) SetSeconds(v any) *Dayjs { return d.setter("seconds", v) }

// Minutes is minutes() (pluralGetSet plugin).
func (d *Dayjs) Minutes() float64 { return d.num("minutes") }

// SetMinutes is minutes(value) (pluralGetSet plugin).
func (d *Dayjs) SetMinutes(v any) *Dayjs { return d.setter("minutes", v) }

// Hours is hours() (pluralGetSet plugin).
func (d *Dayjs) Hours() float64 { return d.num("hours") }

// SetHours is hours(value) (pluralGetSet plugin).
func (d *Dayjs) SetHours(v any) *Dayjs { return d.setter("hours", v) }

// Days is days() (pluralGetSet plugin).
func (d *Dayjs) Days() float64 { return d.num("days") }

// SetDays is days(value) (pluralGetSet plugin).
func (d *Dayjs) SetDays(v any) *Dayjs { return d.setter("days", v) }

// IsoWeeks is isoWeeks() (pluralGetSet plugin).
func (d *Dayjs) IsoWeeks() float64 { return d.num("isoWeeks") }

// SetIsoWeeks is isoWeeks(value) (pluralGetSet plugin).
func (d *Dayjs) SetIsoWeeks(v any) *Dayjs { return d.setter("isoWeeks", v) }

// Months is months() (pluralGetSet plugin).
func (d *Dayjs) Months() float64 { return d.num("months") }

// SetMonths is months(value) (pluralGetSet plugin).
func (d *Dayjs) SetMonths(v any) *Dayjs { return d.setter("months", v) }

// Quarters is quarters() (pluralGetSet plugin).
func (d *Dayjs) Quarters() float64 { return d.num("quarters") }

// SetQuarters is quarters(value) (pluralGetSet plugin).
func (d *Dayjs) SetQuarters(v any) *Dayjs { return d.setter("quarters", v) }

// Years is years() (pluralGetSet plugin).
func (d *Dayjs) Years() float64 { return d.num("years") }

// SetYears is years(value) (pluralGetSet plugin).
func (d *Dayjs) SetYears(v any) *Dayjs { return d.setter("years", v) }

// Dates is dates() (pluralGetSet plugin).
func (d *Dayjs) Dates() float64 { return d.num("dates") }

// SetDates is dates(value) (pluralGetSet plugin).
func (d *Dayjs) SetDates(v any) *Dayjs { return d.setter("dates", v) }
