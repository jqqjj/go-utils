package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

var registryInt sync.Map // map[reflect.EnumIntType]any

type EnumInt[T any] struct {
	v   int
	set bool
}

func (e EnumInt[T]) IsSet() bool {
	return e.set
}

func (e EnumInt[T]) GetValue() (int, bool) {
	return e.v, e.set
}

func (e EnumInt[T]) Int() int {
	return e.v
}

func (e EnumInt[T]) MarshalJSON() ([]byte, error) {
	v, ok := e.GetValue()
	if !ok {
		return []byte("null"), nil
	}

	return []byte(strconv.Itoa(v)), nil
}

func (e *EnumInt[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = EnumInt[T]{}
		return nil
	}

	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	typ, ok := EnumIntTypeGet[T]()
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

func (e *EnumInt[T]) Scan(value any) error {
	if value == nil {
		*e = EnumInt[T]{}
		return nil
	}

	var v int
	switch value := value.(type) {
	case int:
		v = value
	case int8:
		v = int(value)
	case int16:
		v = int(value)
	case int32:
		v = int(value)
	case int64:
		if !int64FitsInt(value) {
			return fmt.Errorf("enum int scan value overflows int: %d", value)
		}
		v = int(value)
	case uint:
		if !uint64FitsInt(uint64(value)) {
			return fmt.Errorf("enum int scan value overflows int: %d", value)
		}
		v = int(value)
	case uint8:
		v = int(value)
	case uint16:
		v = int(value)
	case uint32:
		if !uint64FitsInt(uint64(value)) {
			return fmt.Errorf("enum int scan value overflows int: %d", value)
		}
		v = int(value)
	case uint64:
		if !uint64FitsInt(value) {
			return fmt.Errorf("enum int scan value overflows int: %d", value)
		}
		v = int(value)
	case []byte:
		parsed, err := strconv.Atoi(string(value))
		if err != nil {
			return err
		}
		v = parsed
	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		v = parsed
	default:
		return fmt.Errorf("unsupported enum int scan type: %T", value)
	}

	typ, ok := EnumIntTypeGet[T]()
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

func int64FitsInt(v int64) bool {
	maxInt := int64(^uint(0) >> 1)
	minInt := -maxInt - 1
	return v >= minInt && v <= maxInt
}

func uint64FitsInt(v uint64) bool {
	maxInt := uint64(^uint(0) >> 1)
	return v <= maxInt
}

func (e EnumInt[T]) Value() (driver.Value, error) {
	if !e.set {
		return nil, nil
	}

	return int64(e.v), nil
}

type EnumIntType[T any] struct {
	mu      sync.RWMutex
	values  map[int]struct{}
	members []EnumInt[T]
}

func NewEnumIntType[T any]() *EnumIntType[T] {
	t := &EnumIntType[T]{values: make(map[int]struct{})}

	key := reflect.TypeOf((*T)(nil)).Elem()
	if _, loaded := registryInt.LoadOrStore(key, t); loaded {
		panic(fmt.Sprintf("enum type already registered: %v", key))
	}

	return t
}

func EnumIntTypeGet[T any]() (*EnumIntType[T], bool) {
	v, ok := registryInt.Load(reflect.TypeOf((*T)(nil)).Elem())
	if !ok {
		return nil, false
	}

	t, ok := v.(*EnumIntType[T])
	return t, ok
}

func (t *EnumIntType[T]) Add(v int) EnumInt[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.values[v]; ok {
		panic(fmt.Sprintf("enum value duplicated: %d", v))
	}

	t.values[v] = struct{}{}

	e := EnumInt[T]{
		v:   v,
		set: true,
	}
	t.members = append(t.members, e)

	return e
}

func (t *EnumIntType[T]) Parse(v int) (EnumInt[T], error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.values[v]; !ok {
		return EnumInt[T]{}, fmt.Errorf("invalid enum value: %d", v)
	}

	return EnumInt[T]{
		v:   v,
		set: true,
	}, nil
}

func (t *EnumIntType[T]) Members() []EnumInt[T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	members := make([]EnumInt[T], len(t.members))
	copy(members, t.members)

	return members
}

func (t *EnumIntType[T]) IsAny(v EnumInt[T], values ...EnumInt[T]) bool {
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
