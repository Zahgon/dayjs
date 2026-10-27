package icu

import (
	"math"
	"sort"
)

// transitionsBetween returns the offset in effect at epoch second from, and the zone's
// transitions [epochSecond, newOffset] with from < t <= to, in order.
func (z *tzZone) transitionsBetween(from, to int64) (int64, [][2]int64) {
	start := z.offsetAt(from)
	y2100 := tzdata().Y2100
	var out [][2]int64
	if from < y2100 {
		i := sort.Search(len(z.Trans), func(i int) bool { return z.Trans[i][0] > from })
		for ; i < len(z.Trans) && z.Trans[i][0] <= to; i++ {
			out = append(out, z.Trans[i])
		}
	}
	if to < y2100 || (z.Rule == nil && z.Cycle == nil) {
		return start, out
	}
	lo := from
	if lo < y2100 {
		lo = y2100 - 1
	}
	var extra [][2]int64
	if z.Rule != nil {
		base := z.baseAt2100()
		y0, _, _ := civilFromDays(floorDiv(lo+base, 86400))
		y1, _, _ := civilFromDays(floorDiv(to+base, 86400))
		prev := z.Rule[len(z.Rule)-1][4]
		for yy := y0 - 1; yy <= y1+1; yy++ {
			for _, t := range ruleTransitions(z.Rule, yy, prev) {
				if t[0] >= y2100 && t[0] > lo && t[0] <= to {
					extra = append(extra, t)
				}
			}
		}
	} else {
		k0 := floorDiv(lo-y2100, cycleSeconds)
		k1 := floorDiv(to-y2100, cycleSeconds)
		for k := k0; k <= k1; k++ {
			for _, c := range z.Cycle {
				t := c[0] + k*cycleSeconds
				if t >= y2100 && t > lo && t <= to {
					extra = append(extra, [2]int64{t, c[1]})
				}
			}
		}
	}
	sort.SliceStable(extra, func(i, j int) bool { return extra[i][0] < extra[j][0] })
	return start, append(out, extra...)
}

// maxOffsetSeconds bounds any zone offset; ICU uses one day.
const maxOffsetSeconds = 86400

// OffsetFromLocal returns the UTC offset, in seconds, that ICU's
// BasicTimeZone::getOffsetFromLocal gives a wall-clock time localMs (milliseconds as if UTC) with
// both options UCAL_TZ_LOCAL_FORMER, which is how V8 converts local time to UTC. A skipped wall
// time takes the offset before the transition, and so does a repeated one.
func OffsetFromLocal(canonical string, localMs float64) int64 {
	if off, ok := parseOffsetID(canonical); ok {
		return off
	}
	z := tzdata().Zones[canonical]
	if z == nil {
		return 0
	}
	sec := int64(math.Floor(localMs / 1000))
	before, trans := z.transitionsBetween(sec-2*maxOffsetSeconds, sec+2*maxOffsetSeconds)
	for i := len(trans) - 1; i >= 0; i-- {
		prev := before
		if i > 0 {
			prev = trans[i-1][1]
		}
		after := trans[i][1]
		local := trans[i][0]
		if after-prev >= 0 {
			local += after
		} else {
			local += prev
		}
		if sec >= local {
			return after
		}
	}
	return before
}

// ZoneOffset returns the UTC offset, in seconds, of a zone at the given UTC time in milliseconds.
func ZoneOffset(canonical string, ms float64) int64 {
	return OffsetSeconds(canonical, int64(math.Floor(ms/1000)))
}
