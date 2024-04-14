package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
)

type JsonNullString sql.NullString

func (j *JsonNullString) Scan(value interface{}) (err error) {
	if value == nil {
		j.String, j.Valid = "", false
		return
	}

	switch value.(type) {
	case string:
		j.String = value.(string)
	case []uint8:
		j.String = string(value.([]uint8))
	default:
		return errors.New("invalid value")
	}

	j.Valid = true
	return
}

func (j JsonNullString) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.String, nil
}

func (j JsonNullString) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(j.String)
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullString) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		j.Valid = true
		j.String = *x
	} else {
		j.Valid = false
	}
	return nil
}

func (j JsonNullString) EncodeValues(key string, val *url.Values) error {
	if j.Valid {
		val.Set(key, j.String)
	} else {
		val.Set(key, "")
	}
	return nil
}
