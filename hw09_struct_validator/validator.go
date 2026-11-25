package hw09structvalidator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotStruct       = errors.New("value must be a struct")
	ErrUnknownRule     = errors.New("unknown validation rule")
	ErrInvalidRule     = errors.New("invalid validation rule")
	ErrUnsupportedType = errors.New("unsupported field type")
	ErrInvalidRegexp   = errors.New("invalid regular expression")

	ErrLen    = errors.New("length validation failed")
	ErrRegexp = errors.New("regexp validation failed")
	ErrIn     = errors.New("in validation failed")
	ErrMin    = errors.New("min validation failed")
	ErrMax    = errors.New("max validation failed")
)

type ValidationError struct {
	Field string
	Err   error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	messages := make([]string, 0, len(v))
	for _, validationError := range v {
		messages = append(messages, fmt.Sprintf("%s: %v", validationError.Field, validationError.Err))
	}

	return strings.Join(messages, "; ")
}

func Validate(v interface{}) error {
	value, err := structValue(v)
	if err != nil {
		return err
	}

	valueType := value.Type()
	var validationErrors ValidationErrors

	for i := 0; i < value.NumField(); i++ {
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		tag, ok := structField.Tag.Lookup("validate")
		if !ok || tag == "" {
			continue
		}

		fieldErrors, err := validateField(structField.Name, value.Field(i), tag)
		if err != nil {
			return err
		}

		validationErrors = append(validationErrors, fieldErrors...)
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func structValue(v interface{}) (reflect.Value, error) {
	value := reflect.ValueOf(v)
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}, ErrNotStruct
		}
		value = value.Elem()
	}

	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, ErrNotStruct
	}

	return value, nil
}

type rule struct {
	name  string
	param string
}

func parseRules(tag string) ([]rule, error) {
	parts := strings.Split(tag, "|")
	rules := make([]rule, 0, len(parts))

	for _, part := range parts {
		name, param, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrInvalidRule, part)
		}

		rules = append(rules, rule{name: name, param: param})
	}

	return rules, nil
}

func validateField(name string, value reflect.Value, tag string) (ValidationErrors, error) {
	rules, err := parseRules(tag)
	if err != nil {
		return nil, err
	}

	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		return validateSlice(name, value, rules)
	}

	return validateScalar(name, value, rules)
}

func validateSlice(name string, value reflect.Value, rules []rule) (ValidationErrors, error) {
	validationErrors := make(ValidationErrors, 0, len(rules)*value.Len())

	for i := 0; i < value.Len(); i++ {
		elementErrors, err := validateScalar(name, value.Index(i), rules)
		if err != nil {
			return nil, err
		}

		for _, elementError := range elementErrors {
			elementError.Err = fmt.Errorf("index %d: %w", i, elementError.Err)
			validationErrors = append(validationErrors, elementError)
		}
	}

	return validationErrors, nil
}

func validateScalar(name string, value reflect.Value, rules []rule) (ValidationErrors, error) {
	validationErrors := make(ValidationErrors, 0, len(rules))

	for _, currentRule := range rules {
		err := validateValue(value, currentRule)
		if err == nil {
			continue
		}
		if !isValidationError(err) {
			return nil, err
		}

		validationErrors = append(validationErrors, ValidationError{Field: name, Err: err})
	}

	return validationErrors, nil
}

func validateValue(value reflect.Value, currentRule rule) error {
	switch value.Kind() { //nolint:exhaustive // only strings and integers are supported
	case reflect.String:
		return validateString(value.String(), currentRule)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return validateInt(value.Int(), currentRule)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedType, value.Kind())
	}
}

func validateString(value string, currentRule rule) error {
	switch currentRule.name {
	case "len":
		expected, err := parseLength(currentRule)
		if err != nil {
			return err
		}

		actual := utf8.RuneCountInString(value)
		if actual != expected {
			return fmt.Errorf("%w: %q has %d symbols, want %d", ErrLen, value, actual, expected)
		}
	case "regexp":
		expression, err := regexp.Compile(currentRule.param)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidRegexp, err.Error())
		}
		if !expression.MatchString(value) {
			return fmt.Errorf("%w: %q does not match %q", ErrRegexp, value, currentRule.param)
		}
	case "in":
		if currentRule.param == "" {
			return fmt.Errorf("%w: empty in list", ErrInvalidRule)
		}

		for _, allowed := range strings.Split(currentRule.param, ",") {
			if value == allowed {
				return nil
			}
		}

		return fmt.Errorf("%w: %q is not in %q", ErrIn, value, currentRule.param)
	default:
		return fmt.Errorf("%w: %q for string", ErrUnknownRule, currentRule.name)
	}

	return nil
}

func validateInt(value int64, currentRule rule) error {
	switch currentRule.name {
	case "min":
		bound, err := parseInteger(currentRule)
		if err != nil {
			return err
		}
		if value < bound {
			return fmt.Errorf("%w: %d is less than %d", ErrMin, value, bound)
		}
	case "max":
		bound, err := parseInteger(currentRule)
		if err != nil {
			return err
		}
		if value > bound {
			return fmt.Errorf("%w: %d is greater than %d", ErrMax, value, bound)
		}
	case "in":
		allowed, err := parseIntegerList(currentRule)
		if err != nil {
			return err
		}

		for _, candidate := range allowed {
			if value == candidate {
				return nil
			}
		}

		return fmt.Errorf("%w: %d is not in %v", ErrIn, value, allowed)
	default:
		return fmt.Errorf("%w: %q for int", ErrUnknownRule, currentRule.name)
	}

	return nil
}

func parseLength(currentRule rule) (int, error) {
	length, err := strconv.Atoi(currentRule.param)
	if err != nil || length < 0 {
		return 0, fmt.Errorf("%w: %s:%s", ErrInvalidRule, currentRule.name, currentRule.param)
	}

	return length, nil
}

func parseInteger(currentRule rule) (int64, error) {
	value, err := strconv.ParseInt(currentRule.param, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s:%s", ErrInvalidRule, currentRule.name, currentRule.param)
	}

	return value, nil
}

func parseIntegerList(currentRule rule) ([]int64, error) {
	if currentRule.param == "" {
		return nil, fmt.Errorf("%w: %s:%s", ErrInvalidRule, currentRule.name, currentRule.param)
	}

	parts := strings.Split(currentRule.param, ",")
	values := make([]int64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s:%s", ErrInvalidRule, currentRule.name, currentRule.param)
		}

		values = append(values, value)
	}

	return values, nil
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrLen) ||
		errors.Is(err, ErrRegexp) ||
		errors.Is(err, ErrIn) ||
		errors.Is(err, ErrMin) ||
		errors.Is(err, ErrMax)
}
