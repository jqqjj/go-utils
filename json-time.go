package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"
)

type JsonTime struct {
	time.Time
}

func (j *JsonTime) Scan(value interface{}) error {
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

func (j JsonTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if j.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return j.Time.Format(time.DateTime), nil
}

func (j JsonTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.DateTime)+2)
	b = append(b, '"')
	b = j.AppendFormat(b, time.DateTime)
	b = append(b, '"')
	return b, nil
}

func (j *JsonTime) UnmarshalJSON(data []byte) (err error) {
	j.Time, err = time.ParseInLocation(`"`+time.DateTime+`"`, string(data), time.Local)
	return
}

func (j JsonTime) EncodeValues(key string, val *url.Values) error {
	val.Set(key, j.Time.Format(time.DateTime))
	return nil
}
