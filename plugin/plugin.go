// Package plugin lists the 37 Day.js plugins. Each variable is the plugin of the same name,
// with its first letter capitalized: dayjs.Extend(plugin.CustomParseFormat) is
// dayjs.extend(require('dayjs/plugin/customParseFormat')).
package plugin

import (
	dayjs "github.com/iamkun/dayjs-go"
	"github.com/iamkun/dayjs-go/internal/registry"
)

func get(name string) *dayjs.Plugin { return registry.Plugins[name].(*dayjs.Plugin) }

// The plugins.
var (
	AdvancedFormat     = get("advancedFormat")
	ArraySupport       = get("arraySupport")
	BadMutable         = get("badMutable")
	BigIntSupport      = get("bigIntSupport")
	BuddhistEra        = get("buddhistEra")
	Calendar           = get("calendar")
	CustomParseFormat  = get("customParseFormat")
	DayOfYear          = get("dayOfYear")
	DevHelper          = get("devHelper")
	Duration           = get("duration")
	IsBetween          = get("isBetween")
	IsLeapYear         = get("isLeapYear")
	IsMoment           = get("isMoment")
	IsSameOrAfter      = get("isSameOrAfter")
	IsSameOrBefore     = get("isSameOrBefore")
	IsToday            = get("isToday")
	IsTomorrow         = get("isTomorrow")
	IsYesterday        = get("isYesterday")
	IsoWeek            = get("isoWeek")
	IsoWeeksInYear     = get("isoWeeksInYear")
	LocaleData         = get("localeData")
	LocalizedFormat    = get("localizedFormat")
	MinMax             = get("minMax")
	NegativeYear       = get("negativeYear")
	ObjectSupport      = get("objectSupport")
	PluralGetSet       = get("pluralGetSet")
	PreParsePostFormat = get("preParsePostFormat")
	QuarterOfYear      = get("quarterOfYear")
	RelativeTime       = get("relativeTime")
	Timezone           = get("timezone")
	ToArray            = get("toArray")
	ToObject           = get("toObject")
	UpdateLocale       = get("updateLocale")
	Utc                = get("utc")
	WeekOfYear         = get("weekOfYear")
	WeekYear           = get("weekYear")
	Weekday            = get("weekday")
)
