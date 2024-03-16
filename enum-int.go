package utils

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
)

type EnumInt[T ~int] struct {
	valid bool
	value int
}

func (e *EnumInt[T]) GetValue() int {
	return e.value
}

func (e *EnumInt[T]) Is(a EnumInt[T]) bool {
	return e.valid == a.valid && e.value == a.value
}

func (e *EnumInt[T]) Setted() bool {
	return e.valid
}

func (e EnumInt[T]) MarshalJSON() ([]byte, error) {
	if e.valid {
		return json.Marshal(e.value)
	} else {
		return json.Marshal(nil)
	}
}

func (e *EnumInt[T]) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *int
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

func (e *EnumInt[T]) Scan(value interface{}) error {
	switch value.(type) {
	case int64:
		e.value = int(value.(int64))
	case int32:
		e.value = int(value.(int32))
	case int:
		e.value = value.(int)
	case uint8:
		e.value = int(value.(uint8))
	case []uint8:
		if i, err := strconv.Atoi(string(value.([]uint8))); err != nil {
			return err
		} else {
			e.value = i
		}
	case string:
		if i, err := strconv.Atoi(value.(string)); err != nil {
			return err
		} else {
			e.value = i
		}
	default:
		return errors.New("invalid value")
	}
	e.valid = true
	return nil
}

func (e EnumInt[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return int64(e.value), nil
}

func NewEnumInt[T ~int](value int) EnumInt[T] {
	return EnumInt[T]{valid: true, value: value}
}
