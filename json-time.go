package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type JsonTime struct {
	time.Time
}

func (t *JsonTime) Scan(value interface{}) error {
	switch value.(type) {
	case time.Time:
		t.Time = value.(time.Time)
		return nil
	case string:
		if len(value.(string)) == 0 {
			t.Time = time.Time{}
			return nil
		} else {
			return t.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 {
			t.Time = time.Time{}
			return nil
		} else {
			return t.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (t JsonTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	format := "2006-01-02 15:04:05"
	b := make([]byte, 0, len(format)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, format)
	b = append(b, '"')
	return b, nil
}

func (t *JsonTime) UnmarshalJSON(data []byte) (err error) {
	format := "2006-01-02 15:04:05"
	t.Time, err = time.ParseInLocation(`"`+format+`"`, string(data), time.Local)
	return
}
