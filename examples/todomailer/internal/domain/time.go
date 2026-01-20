package domain

import "time"

// TimeService provides the current time.
type TimeService interface {
	Now() time.Time
}
