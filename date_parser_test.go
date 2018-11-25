package goja

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {

	tst := func(layout, value string, expectedTs int64) func(t *testing.T) {
		return func(t *testing.T) {
			t.Parallel()
			tm, err := parseDate(layout, value, time.UTC)
			if err != nil {
				t.Fatal(err)
			}
			if tm.Unix() != expectedTs {
				t.Fatal(tm)
			}
		}
	}

	t.Run("1", tst("2006-01-02T15:04:05.000Z070000", "2006-01-02T15:04:05.000+07:00:00", 1136189045))
	t.Run("2", tst("2006-01-02T15:04:05.000Z07:00:00", "2006-01-02T15:04:05.000+07:00:00", 1136189045))
	t.Run("3", tst("2006-01-02T15:04:05.000Z07:00", "2006-01-02T15:04:05.000+07:00", 1136189045))
	t.Run("4", tst("2006-01-02T15:04:05.000Z070000", "2006-01-02T15:04:05.000+070000", 1136189045))
	t.Run("5", tst("2006-01-02T15:04:05.000Z070000", "2006-01-02T15:04:05.000Z", 1136214245))
	t.Run("6", tst("2006-01-02T15:04:05.000Z0700", "2006-01-02T15:04:05.000Z", 1136214245))
	t.Run("7", tst("2006-01-02T15:04:05.000Z07", "2006-01-02T15:04:05.000Z", 1136214245))

}
