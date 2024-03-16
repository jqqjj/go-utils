package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JsonNullDate sql.NullTime

func (j *JsonNullDate) Scan(value interface{}) error {
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
		return fmt.Errorf("can not convert %v to date", value)
	}
}

func (j JsonNullDate) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.Time, nil
}

func (j JsonNullDate) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return json.Marshal(nil)
	}
	format := "2006-01-02"
	b := make([]byte, 0, len(format)+2)
	b = append(b, '"')
	b = j.Time.AppendFormat(b, format)
	b = append(b, '"')
	return b, nil
}

func (j *JsonNullDate) UnmarshalJSON(data []byte) (err error) {
	var s *string
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		j.Valid = false
		return
	}
	if j.Time, err = time.ParseInLocation("2006-01-02", *s, time.Local); err != nil {
		return err
	}
	j.Valid = true
	return
}
