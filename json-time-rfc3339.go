package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"
)

type JsonTimeRFC3339 time.Time

func (j *JsonTimeRFC3339) Scan(value interface{}) error {
	switch value.(type) {
	case time.Time:
		*j = JsonTimeRFC3339(value.(time.Time))
		return nil
	case string:
		if len(value.(string)) == 0 {
			*j = JsonTimeRFC3339(time.Time{})
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 {
			*j = JsonTimeRFC3339(time.Time{})
			return nil
		} else {
			return j.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (j JsonTimeRFC3339) Value() (driver.Value, error) {
	var zeroTime time.Time
	if time.Time(j).UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return time.Time(j), nil
}

func (j JsonTimeRFC3339) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.RFC3339)+2)
	b = append(b, '"')
	b = time.Time(j).AppendFormat(b, time.RFC3339)
	b = append(b, '"')
	return b, nil
}

func (j *JsonTimeRFC3339) UnmarshalJSON(data []byte) (err error) {
	var tmpTime time.Time
	tmpTime, err = time.Parse(`"`+time.RFC3339+`"`, string(data))
	*j = JsonTimeRFC3339(tmpTime.Local())
	return
}

func (j JsonTimeRFC3339) String() string {
	return time.Time(j).String()
}

func (j JsonTimeRFC3339) EncodeValues(key string, val *url.Values) error {
	val.Set(key, time.Time(j).Format(time.RFC3339))
	return nil
}
