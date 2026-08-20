package driven

import "time"

type DefaultTimeProvider struct{}

func (tp DefaultTimeProvider) Time() (time.Time, error) {
	return time.Now(), nil
}
