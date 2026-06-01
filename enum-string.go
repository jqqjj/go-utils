package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

var registryString sync.Map // map[reflect.Type]any

type EnumString[T any] struct {
	v   string
	set bool
}

func (e EnumString[T]) IsSet() bool {
	return e.set
}

func (e EnumString[T]) String() (string, bool) {
	return e.v, e.set
}

func (e EnumString[T]) MarshalJSON() ([]byte, error) {
	v, ok := e.String()
	if !ok {
		return []byte("null"), nil
	}

	return json.Marshal(v)
}

func (e *EnumString[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = EnumString[T]{}
		return nil
	}

	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	typ, ok := EnumStringTypeGet[T]()
	if !ok {
		return fmt.Errorf("enum type not registered")
	}

	parsed, err := typ.Parse(v)
	if err != nil {
		return err
	}

	*e = parsed
	return nil
}

func (e *EnumString[T]) Scan(value any) error {
	if value == nil {
		*e = EnumString[T]{}
		return nil
	}

	var v string
	switch value := value.(type) {
	case string:
		v = value
	case []byte:
		v = string(value)
	default:
		return fmt.Errorf("unsupported enum string scan type: %T", value)
	}

	typ, ok := EnumStringTypeGet[T]()
	if !ok {
		return fmt.Errorf("enum type not registered")
	}

	parsed, err := typ.Parse(v)
	if err != nil {
		return err
	}

	*e = parsed
	return nil
}

func (e EnumString[T]) Value() (driver.Value, error) {
	if !e.set {
		return nil, nil
	}

	return e.v, nil
}

type EnumStringType[T any] struct {
	mu     sync.RWMutex
	values map[string]struct{}
}

func NewEnumStringType[T any]() *EnumStringType[T] {
	t := &EnumStringType[T]{
		values: make(map[string]struct{}),
	}

	key := reflect.TypeOf((*T)(nil)).Elem()
	if _, loaded := registryString.LoadOrStore(key, t); loaded {
		panic(fmt.Sprintf("enum type already registered: %v", key))
	}

	return t
}

func EnumStringTypeGet[T any]() (*EnumStringType[T], bool) {
	v, ok := registryString.Load(reflect.TypeOf((*T)(nil)).Elem())
	if !ok {
		return nil, false
	}

	t, ok := v.(*EnumStringType[T])
	return t, ok
}

func (t *EnumStringType[T]) Add(v string) EnumString[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.values[v]; ok {
		panic(fmt.Sprintf("enum value duplicated: %q", v))
	}

	t.values[v] = struct{}{}

	return EnumString[T]{
		v:   v,
		set: true,
	}
}

func (t *EnumStringType[T]) Parse(v string) (EnumString[T], error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.values[v]; !ok {
		return EnumString[T]{}, fmt.Errorf("invalid enum value: %q", v)
	}

	return EnumString[T]{
		v:   v,
		set: true,
	}, nil
}

func (t *EnumStringType[T]) IsAny(v EnumString[T], values ...EnumString[T]) bool {
	if !v.set {
		return false
	}

	for _, value := range values {
		if value.set && value.v == v.v {
			return true
		}
	}

	return false
}
