package ytbapi

import (
	"time"

	"github.com/pkg/errors"
)

func parseDate(s string) (time.Time, error) {
	date, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to parse date: %s", s)
	}
	return date, nil
}
