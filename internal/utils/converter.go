package utils

import (
	"time"

	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertTimestampToTime конвертация из *timestamp.Timestamp в time.Time.
func ConvertTimestampToTime(ts *timestamp.Timestamp) (time.Time, error) {
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, err
	}
	return ts.AsTime(), nil
}
