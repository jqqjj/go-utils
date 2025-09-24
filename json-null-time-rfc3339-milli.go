package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type JsonNullTimeRFC3339Milli sql.NullTime

func (j *JsonNullTimeRFC3339Milli) Scan(value interface{}) error {
	if value == nil {
		j.Time, j.Valid = time.Time{}, false
		return nil
	}
	switch value.(type) {
	case time.Time:
		j.Time, j.Valid = value.(time.Time), true
		return nil
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			j.Time, j.Valid = time.Time{}, false
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 || string(value.([]byte)) == "null" {
			j.Time, j.Valid = time.Time{}, false
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (j JsonNullTimeRFC3339Milli) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.Time.Format("2006-01-02T15:04:05.000Z07:00"), nil
}

func (j JsonNullTimeRFC3339Milli) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return json.Marshal(nil)
	}
	b := make([]byte, 0, len("2006-01-02T15:04:05.000Z07:00")+2)
	b = append(b, '"')
	b = j.Time.AppendFormat(b, "2006-01-02T15:04:05.000Z07:00")
	b = append(b, '"')
	return b, nil
}

func (j *JsonNullTimeRFC3339Milli) UnmarshalJSON(data []byte) (err error) {
	var s *string
	if err = json.Unmarshal(data, &s); err != nil {
		return
	}
	if s == nil {
		j.Valid = false
		return
	}
	str := *s
	// 去掉外层引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	// 零值处理
	if str == "null" || str == "" || strings.HasPrefix(str, "0000-00-00") {
		j.Valid = false
		return
	}
	j.Time, err = time.ParseInLocation("2006-01-02T15:04:05.000Z07:00", str, time.Local)
	j.Valid = err == nil
	return
}

func (j JsonNullTimeRFC3339Milli) EncodeValues(key string, val *url.Values) error {
	if j.Valid {
		val.Set(key, j.Time.Format("2006-01-02T15:04:05.000Z07:00"))
	} else {
		val.Set(key, "")
	}
	return nil
}
