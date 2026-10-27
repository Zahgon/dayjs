// Package icu reproduces the time zone behaviour that Node.js 24 takes from its bundled ICU 78.3
// (tz data 2026e): UTC offsets of every zone at any instant, the resolution of local wall-clock
// times that V8 asks ICU for, time zone identifiers as Intl reports them, and the en-US zone names
// of Intl.DateTimeFormat. The tables under data/ were generated from that runtime.
package icu

import (
	"compress/gzip"
	"embed"
	"encoding/json"
)

//go:embed data/*.json.gz
var dataFS embed.FS

func loadJSON(name string, v any) {
	f, err := dataFS.Open("data/" + name + ".json.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(zr).Decode(v); err != nil {
		panic(err)
	}
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }
