package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JsonNullString sql.NullString

func (v *JsonNullString) Scan(value interface{}) (err error) {
	if value == nil {
		v.String, v.Valid = "", false
		return
	}

	switch value.(type) {
	case string:
		v.String = value.(string)
	case []uint8:
		v.String = string(value.([]uint8))
	default:
		return errors.New("invalid value")
	}

	v.Valid = true
	return
}

func (v JsonNullString) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.String, nil
}

func (v JsonNullString) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.String)
	} else {
		return json.Marshal(nil)
	}
}

func (v *JsonNullString) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.String = *x
	} else {
		v.Valid = false
	}
	return nil
}
