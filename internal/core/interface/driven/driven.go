package driven

import "time"

type TimeProvider interface {
	Time() (time.Time, error)
}
