package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"
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
	if j.Time.IsZero() {
		return nil, nil
	}
	return j.Time.Format("2006-01-02 15:04:05.000"), nil
}

func (j JsonTimeMilli) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len("2006-01-02 15:04:05.000")+2)
	b = append(b, '"')
	b = j.Time.AppendFormat(b, "2006-01-02 15:04:05.000")
	b = append(b, '"')
	return b, nil
}

func (j *JsonTimeMilli) UnmarshalJSON(data []byte) (err error) {
	j.Time = time.Time{}
	str := string(data)
	// 去掉外层引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	// 零值处理
	if str == "null" || str == "" || strings.HasPrefix(str, "0000-00-00") {
		return
	}
	j.Time, err = time.ParseInLocation("2006-01-02 15:04:05.000", str, time.Local)
	return
}

func (j JsonTimeMilli) EncodeValues(key string, val *url.Values) error {
	val.Set(key, j.Time.Format("2006-01-02 15:04:05.000"))
	return nil
}
