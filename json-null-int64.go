package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
)

type JsonNullInt64 sql.NullInt64

func (j *JsonNullInt64) Scan(value interface{}) (err error) {
	if value == nil {
		j.Int64, j.Valid = 0, false
		return
	}

	switch value.(type) {
	case uint32:
		j.Int64 = int64(value.(uint32))
	case int32:
		j.Int64 = int64(value.(int32))
	case int64:
		j.Int64 = value.(int64)
	case []uint8:
		if j.Int64, err = strconv.ParseInt(string(value.([]uint8)), 10, 64); err != nil {
			return
		}
	case string:
		if len(value.(string)) == 0 || value.(string) == "null" {
			j.Int64, j.Valid = 0, false
			return
		}
		if j.Int64, err = strconv.ParseInt(value.(string), 10, 64); err != nil {
			return
		}
	default:
		return errors.New("invalid value")
	}

	j.Valid = true
	return
}

func (j JsonNullInt64) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.Int64, nil
}

func (j JsonNullInt64) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(j.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullInt64) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *int64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		j.Valid = true
		j.Int64 = *x
	} else {
		j.Valid = false
	}
	return nil
}

func (j JsonNullInt64) EncodeValues(key string, val *url.Values) error {
	if j.Valid {
		val.Set(key, strconv.FormatInt(j.Int64, 10))
	} else {
		val.Set(key, "")
	}
	return nil
}
