package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"
)

type JsonTimeMilli struct {
	time.Time
}

func (j *JsonTimeMilli) Scan(value interface{}) error {
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

func (j JsonTimeMilli) Value() (driver.Value, error) {
	return j.Time.Format("2006-01-02 15:04:05.000"), nil
}

func (j JsonTimeMilli) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len("2006-01-02 15:04:05.000")+2)
	b = append(b, '"')
	b = j.AppendFormat(b, "2006-01-02 15:04:05.000")
	b = append(b, '"')
	return b, nil
}

func (j *JsonTimeMilli) UnmarshalJSON(data []byte) (err error) {
	j.Time, err = time.ParseInLocation(`"`+"2006-01-02 15:04:05.000"+`"`, string(data), time.Local)
	return
}

func (j JsonTimeMilli) EncodeValues(key string, val *url.Values) error {
	val.Set(key, j.Time.Format("2006-01-02 15:04:05.000"))
	return nil
}
