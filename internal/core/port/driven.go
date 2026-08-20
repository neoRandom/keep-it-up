package port

import "time"

type TimeProvider interface {
	Time() (time.Time, error)
}
