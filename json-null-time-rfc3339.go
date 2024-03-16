package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JsonNullTimeRFC3339 sql.NullTime

func (v *JsonNullTimeRFC3339) Scan(value interface{}) error {
	if value == nil {
		v.Time, v.Valid = time.Time{}, false
		return nil
	}
	switch value.(type) {
	case time.Time:
		v.Time, v.Valid = value.(time.Time), true
		return nil
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			v.Time, v.Valid = time.Time{}, false
			return nil
		} else {
			return v.UnmarshalJSON([]byte(`"` + value.(string) + `"`))
		}
	case []byte:
		if len(value.([]byte)) == 0 || string(value.([]byte)) == "null" {
			v.Time, v.Valid = time.Time{}, false
			return nil
		} else {
			return v.UnmarshalJSON([]byte(`"` + string(value.([]byte)) + `"`))
		}
	default:
		return fmt.Errorf("can not convert %v to timestamp", value)
	}
}

func (v JsonNullTimeRFC3339) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.Time, nil
}

func (v JsonNullTimeRFC3339) MarshalJSON() ([]byte, error) {
	if !v.Valid {
		return json.Marshal(nil)
	}
	b := make([]byte, 0, len(time.RFC3339)+2)
	b = append(b, '"')
	b = v.Time.AppendFormat(b, time.RFC3339)
	b = append(b, '"')
	return b, nil
}

func (v *JsonNullTimeRFC3339) UnmarshalJSON(data []byte) (err error) {
	var s *string
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		v.Valid = false
		return
	}
	if v.Time, err = time.ParseInLocation(time.RFC3339, *s, time.Local); err != nil {
		return err
	}
	v.Valid = true
	return
}
