package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
)

type JsonNullInt64 sql.NullInt64

func (v *JsonNullInt64) Scan(value interface{}) (err error) {
	if value == nil {
		v.Int64, v.Valid = 0, false
		return
	}

	switch value.(type) {
	case uint32:
		v.Int64 = int64(value.(uint32))
	case int32:
		v.Int64 = int64(value.(int32))
	case int64:
		v.Int64 = value.(int64)
	case []uint8:
		if v.Int64, err = strconv.ParseInt(string(value.([]uint8)), 10, 64); err != nil {
			return
		}
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			v.Int64, v.Valid = 0, false
			return
		}
		if v.Int64, err = strconv.ParseInt(value.(string), 10, 64); err != nil {
			return
		}
	default:
		return errors.New("invalid value")
	}

	v.Valid = true
	return
}

func (v JsonNullInt64) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.Int64, nil
}

func (v JsonNullInt64) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (v *JsonNullInt64) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *int64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.Int64 = *x
	} else {
		v.Valid = false
	}
	return nil
}
