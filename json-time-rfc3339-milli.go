package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
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
	tmpTime, err = time.Parse(`"`+"2006-01-02T15:04:05.000Z07:00"+`"`, string(data))
	*j = JsonTimeRFC3339Milli(tmpTime.Local())
	return
}

func (j JsonTimeRFC3339Milli) String() string {
	return time.Time(j).String()
}

func (j JsonTimeRFC3339Milli) EncodeValues(key string, val *url.Values) error {
	val.Set(key, time.Time(j).Format("2006-01-02T15:04:05.000Z07:00"))
	return nil
}
