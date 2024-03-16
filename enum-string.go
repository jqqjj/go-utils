package utils

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type EnumString[T ~string] struct {
	valid bool
	value string
}

func (e *EnumString[T]) GetValue() string {
	return e.value
}

func (e *EnumString[T]) Is(a EnumString[T]) bool {
	return e.valid == a.valid && e.value == a.value
}

func (e *EnumString[T]) Setted() bool {
	return e.valid
}

func (e EnumString[T]) MarshalJSON() ([]byte, error) {
	if e.valid {
		return json.Marshal(e.value)
	} else {
		return json.Marshal(nil)
	}
}

func (e *EnumString[T]) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		e.valid = true
		e.value = *x
	} else {
		e.valid = false
	}
	return nil
}

func (e *EnumString[T]) Scan(value interface{}) error {
	switch value.(type) {
	case string:
		e.value = value.(string)
	case []uint8:
		e.value = string(value.([]byte))
	default:
		return errors.New("invalid value")
	}
	e.valid = true
	return nil
}

func (e EnumString[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return e.value, nil
}

func NewEnumString[T ~string](value string) EnumString[T] {
	return EnumString[T]{valid: true, value: value}
}
