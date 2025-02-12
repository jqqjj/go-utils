package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"
)

type JsonDate struct {
	time.Time
}

func (j *JsonDate) Scan(value interface{}) error {
	switch value.(type) {
	case time.Time:
		j.Time = value.(time.Time)
		return nil
	case string:
		if len(value.(string)) == 0 {
			j.Time = time.Time{}
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 {
			j.Time = time.Time{}
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (j JsonDate) Value() (driver.Value, error) {
	var zeroTime time.Time
	if j.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return j.Time.Format(time.DateOnly), nil
}

func (j JsonDate) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.DateOnly)+2)
	b = append(b, '"')
	b = j.AppendFormat(b, time.DateOnly)
	b = append(b, '"')
	return b, nil
}

func (j *JsonDate) UnmarshalJSON(data []byte) (err error) {
	j.Time, err = time.Parse(`"`+time.DateOnly+`"`, string(data))
	return
}

func (j JsonDate) EncodeValues(key string, val *url.Values) error {
	val.Set(key, j.UTC().Format(time.DateOnly))
	return nil
}
