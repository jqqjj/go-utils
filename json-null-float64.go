package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
)

type JsonNullFloat64 sql.NullFloat64

func (j *JsonNullFloat64) Scan(value interface{}) (err error) {
	if value == nil {
		j.Float64, j.Valid = 0, false
		return
	}

	switch value.(type) {
	case float32:
		j.Float64 = float64(value.(float32))
	case float64:
		j.Float64 = value.(float64)
	case []uint8:
		if j.Float64, err = strconv.ParseFloat(string(value.([]uint8)), 64); err != nil {
			return
		}
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			j.Float64, j.Valid = 0, false
			return
		}
		if j.Float64, err = strconv.ParseFloat(value.(string), 64); err != nil {
			return
		}
	default:
		return errors.New("invalid value")
	}

	j.Valid = true
	return
}

func (j JsonNullFloat64) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.Float64, nil
}

func (j JsonNullFloat64) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(j.Float64)
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullFloat64) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *float64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		j.Valid = true
		j.Float64 = *x
	} else {
		j.Valid = false
	}
	return nil
}

func (j JsonNullFloat64) EncodeValues(key string, val *url.Values) error {
	if j.Valid {
		val.Set(key, strconv.FormatFloat(j.Float64, 'f', -1, 64))
	} else {
		val.Set(key, "")
	}
	return nil
}
