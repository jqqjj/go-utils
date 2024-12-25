package utils

import (
	"fmt"
	"reflect"
)

func SliceUnique[T comparable](arr []T) []T {
	seen := make(map[T]struct{})
	var result []T
	for _, item := range arr {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func SliceExtractStructField[T any, K comparable](arr []T, fieldName string) ([]K, error) {
	var result []K
	for _, item := range arr {
		rValue := reflect.ValueOf(item)
		for rValue.Kind() == reflect.Pointer {
			if rValue.IsNil() {
				break
			}
			rValue = rValue.Elem()
		}
		if rValue.Kind() != reflect.Struct {
			return nil, fmt.Errorf("%s is not struct type", rValue.Kind())
		}
		field := rValue.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found in struct", fieldName)
		}
		if field.Type().String() != reflect.TypeOf((*K)(nil)).Elem().String() {
			return nil, fmt.Errorf("field %s is not of type %T", fieldName, *new(K))
		}
		result = append(result, field.Interface().(K))
	}
	return result, nil
}

func SliceStructIndex[T any, K comparable](data []T, fieldName string) (map[K]T, error) {
	result := make(map[K]T)
	for _, item := range data {
		rValue := reflect.ValueOf(item)
		for rValue.Kind() == reflect.Pointer {
			if rValue.IsNil() {
				break
			}
			rValue = rValue.Elem()
		}
		if rValue.Kind() != reflect.Struct {
			return nil, fmt.Errorf("%s is not struct type", rValue.Kind())
		}
		field := rValue.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found in struct", fieldName)
		}
		if field.Type().String() != reflect.TypeOf((*K)(nil)).Elem().String() {
			return nil, fmt.Errorf("field %s is not of type %T", fieldName, *new(K))
		}
		result[field.Interface().(K)] = item
	}
	return result, nil
}
