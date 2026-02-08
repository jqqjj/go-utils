package utils

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"sync"
)

var enumStringRegistered sync.Map

type EnumString[T ~string] struct {
	valid bool
	value T
}

func (m EnumString[T]) RawValue() (value T) {
	return m.value
}
func (m EnumString[T]) GetValue() (value T, valid bool) {
	return m.value, m.valid
}
func (m *EnumString[T]) SetValue(value T) bool {
	if m.CheckValue(value) {
		m.valid, m.value = true, value
	}
	return m.valid
}
func (m *EnumString[T]) Set(v Enum[T]) bool {
	value, _ := v.GetValue()
	return m.SetValue(value)
}
func (m EnumString[T]) CheckValue(val T) bool {
	if b, ok := enumStringRegistered.Load(reflect.TypeOf(val)); ok {
		for _, v := range b.(*StringBuilder[T]).Members() {
			if rawValue, valid := v.GetValue(); rawValue == val {
				return valid
			}
		}
	}
	return false
}
func (m EnumString[T]) IsValid() bool {
	return m.valid
}
func (e EnumString[T]) Is(other Enum[T]) bool {
	if other == nil {
		return false
	}
	value, valid := other.GetValue()
	return e.valid == valid && e.value == value
}
func (m EnumString[T]) IsAny(others ...Enum[T]) bool {
	for _, v := range others {
		if m.Is(v) {
			return true
		}
	}
	return false
}

func (e EnumString[T]) MarshalJSON() ([]byte, error) {
	if e.valid {
		return json.Marshal(e.value)
	}
	return json.Marshal(nil)
}

func (e *EnumString[T]) UnmarshalJSON(data []byte) error {
	e.valid = false

	var tmp *string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if tmp != nil {
		e.SetValue(T(*tmp))
	}
	return nil
}

func (e *EnumString[T]) Scan(value any) error {
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
		e.SetValue(T(*tmp))
	}
	return nil
}

func (e EnumString[T]) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return string(e.value), nil
}

type StringBuilder[T ~string] struct {
	members map[T]Enum[T]
}

func (b *StringBuilder[T]) Add(val T) *EnumString[T] {
	b.members[val] = &EnumString[T]{value: val, valid: true}
	return &EnumString[T]{value: val, valid: true}
}

func (b *StringBuilder[T]) Parse(val string) Enum[T] {
	var t EnumString[T]
	if m, ok := b.members[T(val)]; ok {
		return m
	}
	return &t
}

func (b *StringBuilder[T]) Members() (members []Enum[T]) {
	for _, v := range b.members {
		members = append(members, v)
	}
	return
}

func NewStringEnum[T ~string]() *StringBuilder[T] {
	b := &StringBuilder[T]{
		members: make(map[T]Enum[T]),
	}
	var typ T
	enumStringRegistered.Store(reflect.TypeOf(typ), b)
	return b
}
