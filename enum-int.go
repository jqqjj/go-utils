package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"unsafe"
)

var enumInt sync.Map

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
	e.valid = false

	var tmp *int
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if tmp != nil {
		var t EnumInt[T]
		if val, ok := enumInt.Load(reflect.ValueOf(&t).Type().Elem().String()); ok {
			for _, v := range val.([]int) {
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

func (e *EnumInt[T]) Scan(value interface{}) error {
	e.valid = false

	var tmp *int
	switch value.(type) {
	case int64:
		*tmp = int(value.(int64))
	case int32:
		*tmp = int(value.(int32))
	case int:
		*tmp = value.(int)
	case uint8:
		*tmp = int(value.(uint8))
	case []uint8:
		if i, err := strconv.Atoi(string(value.([]uint8))); err == nil {
			*tmp = i
		}
	case string:
		if i, err := strconv.Atoi(value.(string)); err == nil {
			*tmp = i
		}
	}

	if tmp != nil {
		var t EnumInt[T]
		if val, ok := enumInt.Load(reflect.ValueOf(&t).Type().Elem().String()); ok {
			for _, v := range val.([]int) {
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

func (e EnumInt[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return int64(e.value), nil
}

func EnumIntParse[T any](value int) (T, error) {
	var target T
	var actual EnumInt[int]

	tTarget := reflect.ValueOf(&target)
	tActual := reflect.ValueOf(&actual)
	//判断内存布局是否一致
	if tActual.Type().Elem().Kind() != tTarget.Type().Elem().Kind() ||
		tActual.Type().Elem().Size() != tTarget.Type().Elem().Size() ||
		tActual.Type().Elem().NumField() != tTarget.Type().Elem().NumField() {
		return target, fmt.Errorf("type mismatch")
	}

	k := tTarget.Type().Elem().String()
	if val, ok := enumInt.LoadOrStore(k, []int{}); ok {
		for _, v := range val.([]int) {
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

func EnumIntParseBytes[T any](value []byte) (T, error) {
	var target T
	var tmp *int
	if err := json.Unmarshal(value, &tmp); err != nil {
		return target, err
	}
	return EnumIntParse[T](*tmp)
}

func NewEnumInt[T ~int](value int) EnumInt[T] {
	var tmp EnumInt[T]
	k := reflect.ValueOf(&tmp).Type().Elem().String()
	val, _ := enumInt.LoadOrStore(k, []int{})
	enumInt.Store(k, append(val.([]int), value))
	return EnumInt[T]{valid: true, value: value}
}
