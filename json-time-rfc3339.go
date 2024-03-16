package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type JsonTimeRFC3339 time.Time

func (t *JsonTimeRFC3339) Scan(value interface{}) error {
	switch value.(type) {
	case time.Time:
		*t = JsonTimeRFC3339(value.(time.Time))
		return nil
	case string:
		if len(value.(string)) == 0 {
			*t = JsonTimeRFC3339(time.Time{})
			return nil
		} else {
			return t.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 {
			*t = JsonTimeRFC3339(time.Time{})
			return nil
		} else {
			return t.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (t JsonTimeRFC3339) Value() (driver.Value, error) {
	var zeroTime time.Time
	if time.Time(t).UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return time.Time(t), nil
}

func (t JsonTimeRFC3339) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.RFC3339)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, time.RFC3339)
	b = append(b, '"')
	return b, nil
}

func (t *JsonTimeRFC3339) UnmarshalJSON(data []byte) (err error) {
	var tmpTime time.Time
	tmpTime, err = time.Parse(`"`+time.RFC3339+`"`, string(data))
	*t = JsonTimeRFC3339(tmpTime.Local())
	return
}

func (t JsonTimeRFC3339) String() string {
	return time.Time(t).String()
}
