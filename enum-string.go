package utils

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"sync"
)

var enumString sync.Map

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
	e.valid = false

	var tmp *string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if tmp != nil {
		var t T
		if val, ok := enumString.Load(reflect.ValueOf(&t).Type().Elem().String()); ok {
			for _, v := range val.([]string) {
				if *tmp == v {
					e.valid = true
					e.value = *tmp
					break
				}
			}
		}
	}

	return nil
}

func (e *EnumString[T]) Scan(value interface{}) error {
	e.valid = false

	var tmp *string
	switch value.(type) {
	case string:
		*tmp = value.(string)
	case []uint8:
		*tmp = string(value.([]byte))
	}

	if tmp != nil {
		var t T
		if val, ok := enumString.Load(reflect.ValueOf(&t).Type().Elem().String()); ok {
			for _, v := range val.([]string) {
				if *tmp == v {
					e.valid = true
					e.value = *tmp
					break
				}
			}
		}
	}

	return nil
}

func (e EnumString[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return e.value, nil
}

func NewEnumString[T ~string](value string) EnumString[T] {
	var tmp T
	tValue := reflect.ValueOf(&tmp)
	val, _ := enumString.LoadOrStore(tValue.Type().Elem().String(), []string{})
	enumString.Store(tValue.Type().Elem().String(), append(val.([]string), value))
	return EnumString[T]{valid: true, value: value}
}
