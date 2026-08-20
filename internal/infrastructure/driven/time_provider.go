package driven

import "time"

type TimeProvider struct{}

func (tp TimeProvider) Time() (time.Time, error) {
	return time.Now(), nil
}
