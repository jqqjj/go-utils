package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JsonNullTime sql.NullTime

func (v *JsonNullTime) Scan(value interface{}) error {
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

func (v JsonNullTime) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.Time, nil
}

func (v JsonNullTime) MarshalJSON() ([]byte, error) {
	if !v.Valid {
		return json.Marshal(nil)
	}
	format := "2006-01-02 15:04:05"
	b := make([]byte, 0, len(format)+2)
	b = append(b, '"')
	b = v.Time.AppendFormat(b, format)
	b = append(b, '"')
	return b, nil
}

func (v *JsonNullTime) UnmarshalJSON(data []byte) (err error) {
	var s *string
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		v.Valid = false
		return
	}
	if v.Time, err = time.ParseInLocation("2006-01-02 15:04:05", *s, time.Local); err != nil {
		return err
	}
	v.Valid = true
	return
}
