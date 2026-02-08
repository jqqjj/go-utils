package utils

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
	"sync"
)

var enumIntRegistered sync.Map

type Enum[T comparable] interface {
	RawValue() (value T)
	GetValue() (value T, valid bool)
	SetValue(value T) bool
	Set(Enum[T]) bool
	CheckValue(value T) bool
	IsValid() bool
	Is(other Enum[T]) bool
	IsAny(others ...Enum[T]) bool
}

type EnumInt[T ~int] struct {
	valid bool
	value T
}

func (m EnumInt[T]) RawValue() (value T) {
	return m.value
}
func (m EnumInt[T]) GetValue() (value T, valid bool) {
	return m.value, m.valid
}
func (m *EnumInt[T]) SetValue(value T) bool {
	if m.CheckValue(value) {
		m.valid, m.value = true, value
	}
	return m.valid
}
func (m *EnumInt[T]) Set(v Enum[T]) bool {
	value, _ := v.GetValue()
	return m.SetValue(value)
}
func (m EnumInt[T]) CheckValue(val T) bool {
	if b, ok := enumIntRegistered.Load(reflect.TypeOf(val)); ok {
		for _, v := range b.(*IntBuilder[T]).Members() {
			if rawValue, valid := v.GetValue(); rawValue == val {
				return valid
			}
		}
	}
	return false
}
func (m EnumInt[T]) IsValid() bool {
	return m.valid
}
func (e EnumInt[T]) Is(other Enum[T]) bool {
	if other == nil {
		return false
	}
	value, valid := other.GetValue()
	return e.valid == valid && e.value == value
}
func (m *EnumInt[T]) IsAny(others ...Enum[T]) bool {
	for _, v := range others {
		if m.Is(v) {
			return true
		}
	}
	return false
}

func (e EnumInt[T]) MarshalJSON() ([]byte, error) {
	if e.valid {
		return json.Marshal(e.value)
	}
	return json.Marshal(nil)
}

func (e *EnumInt[T]) UnmarshalJSON(data []byte) error {
	e.valid = false

	var tmp *int
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if tmp != nil {
		e.SetValue(T(*tmp))
	}
	return nil
}

func (e *EnumInt[T]) Scan(value any) error {
	e.valid = false

	var tmp *int
	switch value.(type) {
	case int64:
		local := int(value.(int64))
		tmp = &local
	case int32:
		local := int(value.(int32))
		tmp = &local
	case int:
		local := value.(int)
		tmp = &local
	case uint8:
		local := int(value.(uint8))
		tmp = &local
	case []uint8:
		if i, err := strconv.Atoi(string(value.([]uint8))); err == nil {
			tmp = &i
		}
	case string:
		if i, err := strconv.Atoi(value.(string)); err == nil {
			tmp = &i
		}
	}

	if tmp != nil {
		e.SetValue(T(*tmp))
	}
	return nil
}

func (e EnumInt[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return int64(e.value), nil
}

type IntBuilder[T ~int] struct {
	members map[T]Enum[T]
}

func (b *IntBuilder[T]) Add(val T) *EnumInt[T] {
	b.members[val] = &EnumInt[T]{value: val, valid: true}
	return &EnumInt[T]{value: val, valid: true}
}

func (b *IntBuilder[T]) Parse(val int) Enum[T] {
	var t EnumInt[T]
	if m, ok := b.members[T(val)]; ok {
		return m
	}
	return &t
}

func (b *IntBuilder[T]) Members() (members []Enum[T]) {
	for _, v := range b.members {
		members = append(members, v)
	}
	return
}

func NewIntEnum[T ~int]() *IntBuilder[T] {
	b := &IntBuilder[T]{
		members: make(map[T]Enum[T]),
	}
	var typ T
	enumIntRegistered.Store(reflect.TypeOf(typ), b)
	return b
}
