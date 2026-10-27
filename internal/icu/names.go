package icu

import (
	"math"
	"sort"
	"sync"
)

type namesZone struct {
	Periods [][2]int64 `json:"periods"`
	Far     [][2]any   `json:"far"`
}

type namesData struct {
	Strings []string              `json:"strings"`
	Vectors [][2]int              `json:"vectors"`
	Zones   map[string]*namesZone `json:"zones"`
}

var (
	namesOnce  sync.Once
	names      namesData
	namesLower map[string]string
)

const (
	namesEnd = 4133980800 // 2101-01-01T00:00:00Z, the end of the scanned range
	// metazoneEnd is ICU's default end of a metazone mapping, 9999-12-31 23:59 UTC; later instants
	// get GMT-format names wherever the name came from a metazone.
	metazoneEnd = 253402300740
)

func namedata() *namesData {
	namesOnce.Do(func() {
		loadJSON("names", &names)
		namesLower = make(map[string]string, len(names.Zones))
		for id := range names.Zones {
			namesLower[asciiLower(id)] = id
		}
	})
	return &names
}

// HasZoneNames reports whether the en-US names of a zone identifier are known.
func HasZoneNames(zone string) bool {
	d := namedata()
	_, ok := namesLower[asciiLower(zone)]
	return ok && d != nil
}

// ZoneName returns the timeZoneName part that an en-US Intl.DateTimeFormat with two-digit
// numeric fields and hour12 false shows for zone at the instant ms: long selects "long",
// otherwise "short". The bool is false when the zone's names are not known.
func ZoneName(zone string, ms float64, long bool) (string, bool) {
	d := namedata()
	canon, ok := CanonicalTimeZone(zone)
	if !ok {
		return "", false
	}
	if off, isOffset := parseOffsetID(canon); isOffset {
		return gmtFormat(off, long), true
	}
	id, ok := namesLower[asciiLower(zone)]
	if !ok {
		if id, ok = namesLower[asciiLower(canon)]; !ok {
			return "", false
		}
	}
	z := d.Zones[id]
	vec := z.vectorAt(canon, int64(math.Floor(ms/1000)))
	slot := 0
	if long {
		slot = 1
	}
	s := d.Vectors[vec][slot]
	if s < 0 {
		return "", false
	}
	return d.Strings[s], true
}

func (z *namesZone) vectorAt(canon string, sec int64) int {
	p := z.Periods
	if sec < p[0][0] {
		return int(p[0][1])
	}
	if sec <= namesEnd {
		i := sort.Search(len(p), func(i int) bool { return p[i][0] > sec }) - 1
		return int(p[i][1])
	}
	off := OffsetSeconds(canon, sec)
	if sec >= metazoneEnd {
		for _, f := range z.Far {
			ms, v := f[0].(float64), f[1]
			if vec, ok := v.(float64); ok && ms/1000 >= metazoneEnd && OffsetSeconds(canon, int64(math.Floor(ms/1000))) == off {
				return int(vec)
			}
		}
	}
	for i := len(p) - 1; i >= 0; i-- {
		if OffsetSeconds(canon, p[i][0]) == off {
			return int(p[i][1])
		}
	}
	return int(p[len(p)-1][1])
}

// gmtFormat is ICU's localized GMT format for en-US: "GMT", "GMT+5:30" (short) or
// "GMT+05:30" (long), with seconds when the offset has them.
func gmtFormat(off int64, long bool) string {
	if off == 0 {
		return "GMT"
	}
	sign := "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	h, m, s := off/3600, off/60%60, off%60
	two := func(n int64) string { return string([]byte{byte('0' + n/10), byte('0' + n%10)}) }
	out := "GMT" + sign
	if long {
		out += two(h) + ":" + two(m)
	} else {
		out += itoa(h)
		if m != 0 || s != 0 {
			out += ":" + two(m)
		}
	}
	if s != 0 {
		out += ":" + two(s)
	}
	return out
}

func itoa(n int64) string {
	if n < 10 {
		return string([]byte{byte('0' + n)})
	}
	return itoa(n/10) + string([]byte{byte('0' + n%10)})
}
