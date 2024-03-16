package utils

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/Eun/go-convert"
	"github.com/iancoleman/strcase"
	"reflect"
	"strings"
	"time"
)

func GormEntityToMap(entity any) (map[string]any, error) {
	rValue := reflect.ValueOf(entity).Elem()

	for rValue.Kind() == reflect.Ptr {
		if rValue.IsNil() {
			rValue.Set(reflect.New(rValue.Type().Elem()))
		}
		rValue = rValue.Elem()
	}

	if rValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid entity param")
	}

	m := make(map[string]any)

	rType := rValue.Type()
	for i := 0; i < rValue.NumField(); i++ {
		vField := rValue.Field(i)
		tField := rType.Field(i)

		tag := strcase.ToSnake(tField.Name)

		tags := strings.Split(tField.Tag.Get("gorm"), ";")
		for _, v := range tags {
			tagInfo := strings.Split(v, ":")
			if len(tagInfo) == 2 && strings.TrimSpace(tagInfo[0]) == "column" {
				if strings.TrimSpace(tagInfo[1]) != "" {
					tag = strings.TrimSpace(tagInfo[1])
				}
				break
			}
		}

		switch vField.Kind() {
		case reflect.Bool:
			fallthrough
		case reflect.Int:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			fallthrough
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			fallthrough
		case reflect.String:
			m[tag] = vField.Interface()

		case reflect.Struct:
			if reflect.PtrTo(vField.Type()).Implements(reflect.TypeOf((*IRepositoryModel)(nil)).Elem()) {
				continue
			}
			if reflect.PtrTo(vField.Type()).Implements(reflect.TypeOf((*driver.Valuer)(nil)).Elem()) {
				scanResult := vField.Addr().MethodByName("Value").Call([]reflect.Value{})
				if len(scanResult) != 2 {
					return nil, fmt.Errorf("fail to get value")
				}
				if !scanResult[1].IsNil() {
					return nil, scanResult[1].Interface().(error)
				}
				m[tag] = scanResult[0].Interface()
			} else {
				return nil, fmt.Errorf("unsupported type: %s", vField.Type())
			}
		}
	}

	return m, nil
}

func GormMapToEntity(m map[string]any, entity any) error {
	rValue := reflect.ValueOf(entity).Elem()
	rType := reflect.TypeOf(entity).Elem()

	for i := 0; i < rValue.NumField(); i++ {
		vField := rValue.Field(i)
		tField := rType.Field(i)

		tag := strcase.ToSnake(tField.Name)

		tags := strings.Split(tField.Tag.Get("gorm"), ";")
		for _, v := range tags {
			tagInfo := strings.Split(v, ":")
			if len(tagInfo) == 2 && strings.TrimSpace(tagInfo[0]) == "column" {
				if strings.TrimSpace(tagInfo[1]) != "" {
					tag = strings.TrimSpace(tagInfo[1])
				}
				break
			}
		}

		valUnknown, ok := m[tag]
		if !ok {
			continue
		}

		for vField.Kind() == reflect.Ptr {
			if vField.IsNil() {
				vField.Set(reflect.New(vField.Type().Elem()))
			}
			vField = vField.Elem()
		}

		switch vField.Kind() {
		case reflect.Bool:
			var boolResult bool
			if err := convert.Convert(valUnknown, &boolResult); err != nil {
				return err
			}
			vField.SetBool(boolResult)
		case reflect.Int:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			var intResult int64
			if err := convert.Convert(valUnknown, &intResult); err != nil {
				return err
			}
			vField.SetInt(intResult)
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			var floatResult float64
			if err := convert.Convert(valUnknown, &floatResult); err != nil {
				return err
			}
			vField.SetFloat(floatResult)
		case reflect.String:
			var stringResult string
			switch valUnknown.(type) {
			case []byte:
				stringResult = string(valUnknown.([]byte))
			case string:
				stringResult = valUnknown.(string)
			default:
				if err := convert.Convert(valUnknown, &stringResult); err != nil {
					return err
				}
			}
			vField.SetString(stringResult)

		case reflect.Struct:
			if reflect.PtrTo(vField.Type()).Implements(reflect.TypeOf((*sql.Scanner)(nil)).Elem()) {
				var scanResult []reflect.Value
				if vField.Type().String() == "gorm.DeletedAt" {
					if valUnknown == nil {
						vField.Set(reflect.Zero(vField.Type()))
						continue
					}
					var t time.Time
					var err error
					switch valUnknown.(type) {
					case time.Time:
						t = valUnknown.(time.Time)
					case string:
						if t, err = time.ParseInLocation("2006-01-02 15:04:05", valUnknown.(string), time.Local); err != nil {
							return err
						}
					default:
						return fmt.Errorf("unsupported type: %s", vField.Type())
					}
					scanResult = vField.Addr().MethodByName("Scan").Call([]reflect.Value{reflect.ValueOf(t)})
					if len(scanResult) == 0 {
						return fmt.Errorf("fail to transfer")
					}
					if !scanResult[0].IsNil() {
						return scanResult[0].Interface().(error)
					}
				} else {
					if valUnknown == nil {
						vField.Set(reflect.Zero(vField.Type()))
						continue
					}
					scanResult = vField.Addr().MethodByName("Scan").Call([]reflect.Value{reflect.ValueOf(valUnknown)})
					if len(scanResult) == 0 {
						return fmt.Errorf("fail to transfer")
					}
					if !scanResult[0].IsNil() {
						return scanResult[0].Interface().(error)
					}
				}
			} else {
				return fmt.Errorf("unsupported type: %s", vField.Type())
			}
		default:
			return fmt.Errorf("unsupported type: %s", vField.Type())
		}
	}
	return nil
}
