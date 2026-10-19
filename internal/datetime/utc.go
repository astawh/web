// Package datetime defines the canonical backend timestamp representation.
package datetime

import "time"

// Layout deliberately fixes precision and the UTC suffix so stored timestamps
// are readable RFC3339 and compare chronologically as text.
const Layout = "2006-01-02T15:04:05Z"

func Format(value time.Time) string { return value.UTC().Format(Layout) }
func Now() string                   { return Format(time.Now()) }
