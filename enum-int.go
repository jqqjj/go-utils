package utils

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
	"sync"
)

var enumIntRegistered sync.Map

type EnumGetter[T comparable] interface {
	GetValue() (bool, T)
}

type EnumInt[T ~int] struct {
	valid bool
	value T
}

func (e EnumInt[T]) GetValue() (bool, T) {
	return e.valid, e.value
}

func (m *EnumInt[T]) Is(other EnumGetter[T]) bool {
	valid, value := other.GetValue()
	return m.value == value && m.valid == valid
}

func (m *EnumInt[T]) IsAny(others []EnumGetter[T]) bool {
	for _, v := range others {
		if m.Is(v) {
			return true
		}
	}
	return false
}

func (m *EnumInt[T]) IsValid() bool {
	return m.valid
}

func (m *EnumInt[T]) RawValue() int {
	return int(m.value)
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
	if tmp == nil {
		return nil
	}

	val := T(*tmp)
	if b, ok := enumIntRegistered.Load(reflect.TypeOf(val)); ok {
		builder := b.(*IntBuilder[T])
		if m, ok := builder.members[val]; ok {
			*e = m
			return nil
		}
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

	if tmp == nil {
		return nil
	}

	val := T(*tmp)
	if b, ok := enumIntRegistered.Load(reflect.TypeOf(val)); ok {
		builder := b.(*IntBuilder[T])
		if m, ok := builder.members[val]; ok {
			*e = m
			return nil
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

type IntBuilder[T ~int] struct {
	members map[T]EnumInt[T]
}

func (b *IntBuilder[T]) Add(val T) EnumInt[T] {
	m := EnumInt[T]{value: val, valid: true}
	b.members[val] = m
	return m
}

func (b *IntBuilder[T]) Parse(val int) EnumInt[T] {
	if m, ok := b.members[T(val)]; ok {
		return m
	}
	return EnumInt[T]{}
}

func (b *IntBuilder[T]) Members() (members []EnumInt[T]) {
	for _, v := range b.members {
		members = append(members, v)
	}
	return
}

func NewIntEnum[T ~int]() *IntBuilder[T] {
	b := &IntBuilder[T]{
		members: make(map[T]EnumInt[T]),
	}
	var typ T
	enumIntRegistered.Store(reflect.TypeOf(typ), b)
	return b
}
