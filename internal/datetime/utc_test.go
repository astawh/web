package datetime_test

import (
	"testing"
	"time"

	"github.com/astawh/web/internal/datetime"
)

func TestFormatNormalizesZonesAndPrecision(t *testing.T) {
	for _, source := range []string{"2026-10-19T16:30:00.123+02:00", "2026-10-19T09:30:00-05:00", "2026-10-19T14:30:00Z"} {
		value, err := time.Parse(time.RFC3339Nano, source)
		if err != nil {
			t.Fatal(err)
		}
		if got := datetime.Format(value); got != "2026-10-19T14:30:00Z" {
			t.Fatalf("%s became %s", source, got)
		}
	}
}
