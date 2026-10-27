package locale_test

import (
	"testing"

	dayjs "github.com/iamkun/dayjs-go"
	"github.com/iamkun/dayjs-go/locale"
)

func TestEveryLocaleLoads(t *testing.T) {
	if len(locale.Names) != 143 {
		t.Fatalf("got %d locales, want 143", len(locale.Names))
	}
	for _, name := range locale.Names {
		o := locale.Load(name)
		if got := o.Get("name"); got != name {
			t.Errorf("locale %s has name %v", name, got)
		}
		if locale.Load(name) != o {
			t.Errorf("locale %s: second load returned a different object", name)
		}
	}
}

func TestLoadRegistersLocally(t *testing.T) {
	locale.Ru()
	if got := dayjs.Locale(); got != "en" {
		t.Fatalf("global locale changed to %v", got)
	}
	d := dayjs.New(1700000000000.0).SetLocale("ru")
	if got := d.Format("D MMMM YYYY, dddd"); got != "15 ноября 2023, среда" && got != "14 ноября 2023, вторник" {
		t.Fatalf("ru format = %q", got)
	}
}
