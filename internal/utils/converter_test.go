package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertTimestampToTime(t *testing.T) {
	t.Run("должен конвертировать timestamp в time", func(t *testing.T) {
		expectedTime := time.Date(
			2026,
			time.June,
			23,
			12,
			30,
			45,
			123456789,
			time.UTC,
		)

		ts := timestamp.New(expectedTime)

		actualTime, err := ConvertTimestampToTime(ts)

		assert.NoError(t, err)
		assert.Equal(t, expectedTime, actualTime)
	})

	t.Run("должен вернуть ошибку при nil timestamp", func(t *testing.T) {
		actualTime, err := ConvertTimestampToTime(nil)

		assert.Error(t, err)
		assert.True(t, actualTime.IsZero())
	})

	t.Run("должен вернуть ошибку при невалидных наносекундах", func(t *testing.T) {
		ts := &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   1_000_000_000,
		}

		actualTime, err := ConvertTimestampToTime(ts)

		assert.Error(t, err)
		assert.True(t, actualTime.IsZero())
	})

	t.Run("должен вернуть ошибку при отрицательных наносекундах", func(t *testing.T) {
		ts := &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   -1,
		}

		actualTime, err := ConvertTimestampToTime(ts)

		assert.Error(t, err)
		assert.True(t, actualTime.IsZero())
	})
}
