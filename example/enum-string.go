package main

import (
	"fmt"

	"github.com/jqqjj/go-utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type EnumString[T any] struct {
	utils.EnumString[T]
}

type EnumStringType[T any] struct {
	base *utils.EnumStringType[T]
}

func NewEnumStringType[T any]() *EnumStringType[T] {
	return &EnumStringType[T]{
		base: utils.NewEnumStringType[T](),
	}
}

func (t *EnumStringType[T]) Add(v string) EnumString[T] {
	return EnumString[T]{
		EnumString: t.base.Add(v),
	}
}

func (t *EnumStringType[T]) Parse(v string) (EnumString[T], error) {
	e, err := t.base.Parse(v)
	if err != nil {
		return EnumString[T]{}, err
	}

	return EnumString[T]{EnumString: e}, nil
}

func (t *EnumStringType[T]) IsAny(v EnumString[T], values ...EnumString[T]) bool {
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

func (e EnumString[T]) MarshalBSONValue() (bsontype.Type, []byte, error) {
	v, ok := e.String()
	if !ok {
		return bson.TypeNull, nil, nil
	}

	return bson.TypeString, bsoncore.AppendString(nil, v), nil
}

func (e *EnumString[T]) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	if t == bson.TypeNull {
		*e = EnumString[T]{}
		return nil
	}

	if t != bson.TypeString {
		return fmt.Errorf("unsupported bson type: %s", t)
	}

	v, _, ok := bsoncore.ReadString(data)
	if !ok {
		return fmt.Errorf("invalid bson string")
	}

	typ, ok := utils.EnumStringTypeGet[T]()
	if !ok {
		return fmt.Errorf("enum type not registered")
	}

	parsed, err := typ.Parse(v)
	if err != nil {
		return err
	}

	e.EnumString = parsed
	return nil
}
