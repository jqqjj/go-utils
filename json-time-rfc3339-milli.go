package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type JsonTimeRFC3339Milli time.Time

func (j *JsonTimeRFC3339Milli) Scan(value interface{}) error {
	switch value.(type) {
	case time.Time:
		*j = JsonTimeRFC3339Milli(value.(time.Time))
		return nil
	case string:
		if len(value.(string)) == 0 {
			*j = JsonTimeRFC3339Milli(time.Time{})
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 {
			*j = JsonTimeRFC3339Milli(time.Time{})
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (j JsonTimeRFC3339Milli) Value() (driver.Value, error) {
	if time.Time(j).IsZero() {
		return nil, nil
	}
	return time.Time(j).Format("2006-01-02T15:04:05.000Z07:00"), nil
}

func (j JsonTimeRFC3339Milli) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len("2006-01-02T15:04:05.000Z07:00")+2)
	b = append(b, '"')
	b = time.Time(j).AppendFormat(b, "2006-01-02T15:04:05.000Z07:00")
	b = append(b, '"')
	return b, nil
}

func (j *JsonTimeRFC3339Milli) UnmarshalJSON(data []byte) (err error) {
	var tmpTime time.Time
	str := string(data)
	// 去掉外层引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	// 零值处理
	if str == "null" || str == "" || strings.HasPrefix(str, "0000-00-00") {
		return
	}
	tmpTime, err = time.ParseInLocation("2006-01-02T15:04:05.000Z07:00", str, time.Local)
	*j = JsonTimeRFC3339Milli(tmpTime)
	return
}

func (j JsonTimeRFC3339Milli) String() string {
	return time.Time(j).String()
}

func (j JsonTimeRFC3339Milli) EncodeValues(key string, val *url.Values) error {
	val.Set(key, time.Time(j).Format("2006-01-02T15:04:05.000Z07:00"))
	return nil
}
