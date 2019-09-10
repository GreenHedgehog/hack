package hack

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

// GetField - return value of `field`,
// return error if:
// (1) - `target` does not contain `field`
// (2) - `target` type is not a pointer to struct or
// interface with underlying type of it.
func GetField(target interface{}, field string) (interface{}, error) {

	targetValue, err := derefStruct(target)
	if err != nil {
		return nil, err
	}

	fieldValue := targetValue.FieldByName(field)
	if !fieldValue.IsValid() {
		return nil, fmt.Errorf("target does not contain field `%s`", field)
	}

	makeSettable(&fieldValue)

	return fieldValue.Interface(), nil
}

// SetField - changes value of `field` with `value`,
// return error if:
// (1) - `target` does not contain `field`
// (2) - `target` type is not a pointer to struct or
// interface with underlying type of it.
// (3) - `value` type is not assignable to `field`
func SetField(target interface{}, field string, value interface{}) error {

	targetValue, err := derefStruct(target)
	if err != nil {
		return err
	}

	fieldValue := targetValue.FieldByName(field)
	if !fieldValue.IsValid() {
		return fmt.Errorf("target does not contain field `%s`", field)
	}

	makeSettable(&fieldValue)

	if value == nil {
		switch fieldValue.Kind() {

		case reflect.Ptr, reflect.Chan, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface:
			fieldType := fieldValue.Type()
			zeroValue := reflect.Zero(fieldType)
			fieldValue.Set(zeroValue)
			return nil

		default:
			return fmt.Errorf(
				`"nil" is not assignable to "%s"`,
				fieldValue.Type(),
			)
		}
	}

	valueType := reflect.ValueOf(value).Type()
	fieldType := fieldValue.Type()
	if !valueType.AssignableTo(fieldType) {
		return fmt.Errorf(
			`"%s" is not assignable to "%s"`,
			valueType,
			fieldType,
		)
	}

	fieldValue.Set(
		reflect.ValueOf(value),
	)

	return nil
}

func derefStruct(target interface{}) (*reflect.Value, error) {

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, errors.New("target must be a pointer to struct")
	}

	value = value.Elem()

	if value.Kind() != reflect.Struct {
		return nil, errors.New("target must be a pointer to struct")
	}

	return &value, nil
}

func makeSettable(field *reflect.Value) {
	if field.CanSet() {
		return
	}

	addr := field.UnsafeAddr()
	*field = reflect.NewAt(
		field.Type(),
		unsafe.Pointer(addr),
	).Elem()
}

func setZeroValue(v *reflect.Value) {
	typ := v.Type()
	zeroValue := reflect.Zero(typ)
	v.Set(zeroValue)
}

// Field - info about struct field
type Field struct {
	Name  string
	Value interface{}
}

// Transform - applies to fn to all struct fields in natural order
func Transform(
	target interface{},
	fn func(Field) (bool, interface{}),
) error {

	targetValue, err := derefStruct(target)
	if err != nil {
		return err
	}

	for i := 0; i < targetValue.NumField(); i++ {

		fieldValue := targetValue.Field(i)
		makeSettable(&fieldValue)

		field := Field{
			Name:  targetValue.Type().Field(i).Name,
			Value: fieldValue.Interface(),
		}

		update, value := fn(field)
		if !update {
			continue
		}

		if value == nil {
			switch fieldValue.Kind() {
			case reflect.Ptr, reflect.Chan, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface:
				setZeroValue(&fieldValue)
				return nil
			default:
				return fmt.Errorf(
					`"nil" is not assignable to "%s"`,
					fieldValue.Type(),
				)
			}
		}

		valueType := reflect.ValueOf(value).Type()
		fieldType := fieldValue.Type()
		if !valueType.AssignableTo(fieldType) {
			return fmt.Errorf(
				`update field "%s" faild: "%s" is not assignable to "%s"`,
				field.Name,
				valueType,
				fieldType,
			)
		}

		fieldValue.Set(
			reflect.ValueOf(value),
		)
	}

	return nil
}
