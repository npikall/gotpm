package eval

import (
	"fmt"
	"strings"
	"time"

	"github.com/npikall/toml-edit/ast"
	"github.com/npikall/toml-edit/token"
)

// LocalDate is a date without time or offset.
type LocalDate struct {
	Year  int
	Month int
	Day   int
}

func (d LocalDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// LocalTime is a time of day without date or offset.
type LocalTime struct {
	Hour       int
	Minute     int
	Second     int
	Nanosecond int
}

func (t LocalTime) String() string {
	s := fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second)
	if t.Nanosecond != 0 {
		s += fmt.Sprintf(".%09d", t.Nanosecond)
	}
	return s
}

// LocalDateTime is a date and time without offset.
type LocalDateTime struct {
	Date LocalDate
	Time LocalTime
}

func (dt LocalDateTime) String() string {
	return dt.Date.String() + "T" + dt.Time.String()
}

// Field layout of the date/time tokens; the lexer guarantees the shape, so
// only ranges are checked here.
const (
	dateLen        = len("2006-01-02")
	minTimeLen     = len("15:04")
	timeWithSecLen = len("15:04:05")
	offsetLen      = len("+07:00")
	maxFracDigits  = 9 // nanoseconds; extra digits are truncated

	maxHour       = 23
	maxMinute     = 59
	maxSecond     = 60 // leap second
	maxMonth      = 12
	secondsPerMin = 60
	decimalBase   = 10
)

func decodeDateTime(s *ast.Scalar) (any, error) {
	raw := s.Raw
	switch s.Type {
	case token.LOCAL_DATE:
		d, ok := parseDate(raw)
		if !ok {
			return nil, errorAt(s.Pos, "invalid date %q", raw)
		}
		return d, nil
	case token.LOCAL_TIME:
		t, ok := parseTime(raw)
		if !ok {
			return nil, errorAt(s.Pos, "invalid time %q", raw)
		}
		return t, nil
	case token.LOCAL_DATETIME:
		d, okDate := parseDate(raw[:dateLen])
		t, okTime := parseTime(raw[dateLen+1:])
		if !okDate || !okTime {
			return nil, errorAt(s.Pos, "invalid date-time %q", raw)
		}
		return LocalDateTime{Date: d, Time: t}, nil
	default: // token.OFFSET_DATETIME
		return decodeOffsetDateTime(s)
	}
}

func decodeOffsetDateTime(s *ast.Scalar) (time.Time, error) {
	raw := s.Raw
	timeEnd := len(raw) - offsetLen
	loc := time.UTC
	if last := raw[len(raw)-1]; last == 'Z' || last == 'z' {
		timeEnd = len(raw) - 1
	} else {
		off := raw[timeEnd:]
		hour, minute := atoi(off[1:3]), atoi(off[4:6])
		if hour > maxHour || minute > maxMinute {
			return time.Time{}, errorAt(s.Pos, "invalid offset in %q", raw)
		}
		secs := (hour*secondsPerMin + minute) * secondsPerMin
		if off[0] == '-' {
			secs = -secs
		}
		loc = time.FixedZone("", secs)
	}
	d, okDate := parseDate(raw[:dateLen])
	t, okTime := parseTime(raw[dateLen+1 : timeEnd])
	if !okDate || !okTime {
		return time.Time{}, errorAt(s.Pos, "invalid date-time %q", raw)
	}
	return time.Date(d.Year, time.Month(d.Month), d.Day, t.Hour, t.Minute, t.Second, t.Nanosecond, loc), nil
}

// parseDate parses "YYYY-MM-DD" and checks month and day ranges.
func parseDate(s string) (LocalDate, bool) {
	d := LocalDate{Year: atoi(s[0:4]), Month: atoi(s[5:7]), Day: atoi(s[8:10])}
	if d.Month < 1 || d.Month > maxMonth || d.Day < 1 || d.Day > daysIn(d.Year, d.Month) {
		return LocalDate{}, false
	}
	return d, true
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// parseTime parses "HH:MM[:SS[.frac]]" and checks hour, minute and second
// ranges.
func parseTime(s string) (LocalTime, bool) {
	t := LocalTime{Hour: atoi(s[0:2]), Minute: atoi(s[3:5])}
	if len(s) > minTimeLen {
		t.Second = atoi(s[6:8])
	}
	if len(s) > timeWithSecLen {
		frac := s[timeWithSecLen+1:]
		if len(frac) > maxFracDigits {
			frac = frac[:maxFracDigits]
		}
		t.Nanosecond = atoi(frac + strings.Repeat("0", maxFracDigits-len(frac)))
	}
	if t.Hour > maxHour || t.Minute > maxMinute || t.Second > maxSecond {
		return LocalTime{}, false
	}
	return t, true
}

// atoi parses a string of ASCII digits.
func atoi(s string) int {
	n := 0
	for i := range len(s) {
		n = n*decimalBase + int(s[i]-'0')
	}
	return n
}
