package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
)

type JsonNullFloat64 sql.NullFloat64

func (v *JsonNullFloat64) Scan(value interface{}) (err error) {
	if value == nil {
		v.Float64, v.Valid = 0, false
		return
	}

	switch value.(type) {
	case float32:
		v.Float64 = float64(value.(float32))
	case float64:
		v.Float64 = value.(float64)
	case []uint8:
		if v.Float64, err = strconv.ParseFloat(string(value.([]uint8)), 64); err != nil {
			return
		}
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			v.Float64, v.Valid = 0, false
			return
		}
		if v.Float64, err = strconv.ParseFloat(value.(string), 64); err != nil {
			return
		}
	default:
		return errors.New("invalid value")
	}

	v.Valid = true
	return
}

func (v JsonNullFloat64) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.Float64, nil
}

func (v JsonNullFloat64) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.Float64)
	} else {
		return json.Marshal(nil)
	}
}

func (v *JsonNullFloat64) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *float64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.Float64 = *x
	} else {
		v.Valid = false
	}
	return nil
}
