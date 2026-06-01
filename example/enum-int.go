package main

import (
	"fmt"

	"github.com/jqqjj/go-utils"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type EnumInt[T any] struct {
	utils.EnumInt[T]
}

type EnumIntType[T any] struct {
	base *utils.EnumIntType[T]
}

func NewEnumIntType[T any]() *EnumIntType[T] {
	return &EnumIntType[T]{
		base: utils.NewEnumIntType[T](),
	}
}

func (t *EnumIntType[T]) Add(v int) EnumInt[T] {
	return EnumInt[T]{
		EnumInt: t.base.Add(v),
	}
}

func (t *EnumIntType[T]) Parse(v int) (EnumInt[T], error) {
	e, err := t.base.Parse(v)
	if err != nil {
		return EnumInt[T]{}, err
	}

	return EnumInt[T]{EnumInt: e}, nil
}

func (t *EnumIntType[T]) IsAny(v EnumInt[T], values ...EnumInt[T]) bool {
	if !v.IsSet() {
		return false
	}

	for _, value := range values {
		if v == value {
			return true
		}
	}

	return false
}

// ---------- BSON ----------

func (e EnumInt[T]) MarshalBSONValue() (bsontype.Type, []byte, error) {
	v, ok := e.Int()
	if !ok {
		return bson.TypeNull, nil, nil
	}

	return bson.TypeInt32, bsoncore.AppendInt32(nil, int32(v)), nil
}

func (e *EnumInt[T]) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	if t == bson.TypeNull {
		*e = EnumInt[T]{}
		return nil
	}

	var v int

	switch t {
	case bson.TypeInt32:
		i, _, ok := bsoncore.ReadInt32(data)
		if !ok {
			return fmt.Errorf("invalid bson int32")
		}
		v = int(i)

	case bson.TypeInt64:
		i, _, ok := bsoncore.ReadInt64(data)
		if !ok {
			return fmt.Errorf("invalid bson int64")
		}
		v = int(i)

	default:
		return fmt.Errorf("unsupported bson type: %s", t)
	}

	typ, ok := utils.EnumIntTypeGet[T]()
	if !ok {
		return fmt.Errorf("enum type not registered")
	}

	parsed, err := typ.Parse(v)
	if err != nil {
		return err
	}

	e.EnumInt = parsed
	return nil
}
