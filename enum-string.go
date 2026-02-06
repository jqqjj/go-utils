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

func (e EnumString[T]) GetValue() (bool, T) {
	return e.valid, e.value
}

func (m *EnumString[T]) Is(other EnumGetter[T]) bool {
	valid, value := other.GetValue()
	return m.value == value && m.valid == valid
}

func (m *EnumString[T]) IsValid() bool {
	return m.valid
}

func (m *EnumString[T]) RawValue() string {
	return string(m.value)
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
	if tmp == nil {
		return nil
	}

	val := T(*tmp)
	if b, ok := enumStringRegistered.Load(reflect.TypeOf(val)); ok {
		builder := b.(*StringBuilder[T])
		if m, ok := builder.members[val]; ok {
			*e = m
			return nil
		}
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

	if tmp == nil {
		return nil
	}

	val := T(*tmp)
	if b, ok := enumStringRegistered.Load(reflect.TypeOf(val)); ok {
		builder := b.(*StringBuilder[T])
		if m, ok := builder.members[val]; ok {
			*e = m
			return nil
		}
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
	members map[T]EnumString[T]
}

func (b *StringBuilder[T]) Add(val T) EnumString[T] {
	m := EnumString[T]{value: val, valid: true}
	b.members[val] = m
	return m
}

func (b *StringBuilder[T]) Parse(val string) EnumString[T] {
	if m, ok := b.members[T(val)]; ok {
		return m
	}
	return EnumString[T]{}
}

func (b *StringBuilder[T]) Members() (members []EnumString[T]) {
	for _, v := range b.members {
		members = append(members, v)
	}
	return
}

func NewStringEnum[T ~string]() *StringBuilder[T] {
	b := &StringBuilder[T]{
		members: make(map[T]EnumString[T]),
	}
	var typ T
	enumStringRegistered.Store(reflect.TypeOf(typ), b)
	return b
}
