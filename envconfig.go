// Package envconfig provides utilities for loading configuration from environment variables
// using struct tags with reflection.
package appkit

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

// LoadFromEnv populates any struct with environment variables based on struct tags.
// Fields should have `env:"ENV_VAR_NAME"` and `default:"default_value"` tags.
func LoadFromEnv(config any) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		envTag := fieldType.Tag.Get("env")
		defaultTag := fieldType.Tag.Get("default")

		if envTag == "" {
			continue
		}

		envValue := os.Getenv(envTag)
		if envValue == "" {
			envValue = defaultTag
		}

		if err := setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("error setting field %s: %v", fieldType.Name, err)
		}
	}
	return nil
}

// setFieldValue sets a struct field value based on its type
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration specifically since it's an int64 under the hood
		if field.Type() == reflect.TypeFor[time.Duration]() {
			if duration, err := time.ParseDuration(value); err == nil {
				field.Set(reflect.ValueOf(duration))
			} else {
				return fmt.Errorf("invalid duration format: %s", value)
			}
		} else {
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(intVal)
			} else {
				return fmt.Errorf("invalid integer format: %s", value)
			}
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			field.SetBool(boolVal)
		} else {
			return fmt.Errorf("invalid boolean format: %s", value)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, field.Type().Bits()); err == nil {
			field.SetFloat(floatVal)
		} else {
			return fmt.Errorf("invalid float format: %s", value)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}
