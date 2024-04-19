package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
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

func (e *EnumString[T]) IsEmpty() bool {
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
		var t EnumString[T]
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
		local := value.(string)
		tmp = &local
	case []uint8:
		local := string(value.([]byte))
		tmp = &local
	}

	if tmp != nil {
		var t EnumString[T]
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

func EnumStringParse[T any](value string) (T, error) {
	var target T
	var actual EnumString[string]

	tTarget := reflect.ValueOf(&target)
	tActual := reflect.ValueOf(&actual)
	//判断内存布局是否一致
	if tActual.Type().Elem().Kind() != tTarget.Type().Elem().Kind() ||
		tActual.Type().Elem().Size() != tTarget.Type().Elem().Size() ||
		tActual.Type().Elem().NumField() != tTarget.Type().Elem().NumField() {
		return target, fmt.Errorf("type mismatch")
	}

	k := tTarget.Type().Elem().String()
	if val, ok := enumString.LoadOrStore(k, []string{}); ok {
		for _, v := range val.([]string) {
			if value == v {
				actual.valid = true
				actual.value = value
				target = *(*T)(unsafe.Pointer(&actual))
				break
			}
		}
	}
	return target, nil
}

func NewEnumString[T ~string](value string) EnumString[T] {
	var tmp EnumString[T]
	k := reflect.ValueOf(&tmp).Type().Elem().String()
	val, _ := enumString.LoadOrStore(k, []string{})
	enumString.Store(k, append(val.([]string), value))
	return EnumString[T]{valid: true, value: value}
}
