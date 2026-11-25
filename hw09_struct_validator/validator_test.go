package hw09structvalidator

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int             `validate:"min:18|max:50"`
		Email  string          `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole        `validate:"in:admin,stuff"`
		Phones []string        `validate:"len:11"`
		meta   json.RawMessage //nolint:unused
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}

	MultiRule struct {
		Value string `validate:"regexp:\\d+|len:2"`
	}

	Numbers struct {
		Min   int `validate:"min:10"`
		Max   int `validate:"max:20"`
		In    int `validate:"in:256,1024"`
		Range int `validate:"min:0|max:10"`
	}

	Ints struct {
		Values []int `validate:"min:1"`
	}

	BadRule struct {
		Value int `validate:"unknown:1"`
	}

	BadLen struct {
		Value string `validate:"len:abc"`
	}

	BadRegexp struct {
		Value string `validate:"regexp:["`
	}

	BadIn struct {
		Value int `validate:"in:1,x"`
	}

	BadType struct {
		Value float64 `validate:"min:1"`
	}

	BadSyntax struct {
		Value int `validate:"min"`
	}

	WithHidden struct {
		hidden  string `validate:"len:99"`
		Visible string `validate:"len:3"`
	}
)

type fieldError struct {
	field  string
	target error
}

type validationTestCase struct {
	name     string
	in       interface{}
	target   error
	expected []fieldError
}

var validationTests = []validationTestCase{
	{
		name: "valid user",
		in:   validUser(),
	},
	{
		name: "invalid user",
		in: User{
			ID:     "short",
			Name:   "Alice",
			Age:    10,
			Email:  "invalid",
			Role:   "user",
			Phones: []string{"123", "12345678901"},
		},
		expected: []fieldError{
			{"ID", ErrLen},
			{"Age", ErrMin},
			{"Email", ErrRegexp},
			{"Role", ErrIn},
			{"Phones", ErrLen},
		},
	},
	{
		name: "valid app",
		in:   App{Version: "1.0.0"},
	},
	{
		name:     "invalid app",
		in:       App{Version: "1.0"},
		expected: []fieldError{{"Version", ErrLen}},
	},
	{
		name: "no validation tags",
		in:   Token{Header: []byte("h"), Payload: []byte("p"), Signature: []byte("s")},
	},
	{
		name: "valid response",
		in:   Response{Code: 200},
	},
	{
		name:     "invalid response",
		in:       Response{Code: 201},
		expected: []fieldError{{"Code", ErrIn}},
	},
	{
		name: "multiple rules are accumulated",
		in:   MultiRule{Value: "abc"},
		expected: []fieldError{
			{"Value", ErrRegexp},
			{"Value", ErrLen},
		},
	},
	{
		name: "valid numbers",
		in:   Numbers{Min: 10, Max: 20, In: 256, Range: 10},
	},
	{
		name: "invalid numbers",
		in:   Numbers{Min: 9, Max: 21, In: 1, Range: 11},
		expected: []fieldError{
			{"Min", ErrMin},
			{"Max", ErrMax},
			{"In", ErrIn},
			{"Range", ErrMax},
		},
	},
	{
		name:     "slice with one invalid element",
		in:       Ints{Values: []int{0, 1, 2}},
		expected: []fieldError{{"Values", ErrMin}},
	},
	{
		name:     "slice with two invalid elements",
		in:       Ints{Values: []int{0, -1, 2}},
		expected: []fieldError{{"Values", ErrMin}, {"Values", ErrMin}},
	},
	{
		name: "empty slice",
		in:   Ints{},
	},
	{
		name: "unexported fields are ignored",
		in:   WithHidden{hidden: "x", Visible: "abc"},
	},
	{
		name: "pointer to valid user",
		in:   ptr(validUser()),
	},
	{
		name:   "not a struct",
		in:     42,
		target: ErrNotStruct,
	},
	{
		name:   "nil",
		in:     nil,
		target: ErrNotStruct,
	},
	{
		name:   "nil struct pointer",
		in:     (*User)(nil),
		target: ErrNotStruct,
	},
	{
		name:   "unknown rule",
		in:     BadRule{Value: 1},
		target: ErrUnknownRule,
	},
	{
		name:   "invalid length parameter",
		in:     BadLen{Value: "x"},
		target: ErrInvalidRule,
	},
	{
		name:   "invalid regexp",
		in:     BadRegexp{Value: "x"},
		target: ErrInvalidRegexp,
	},
	{
		name:   "invalid in parameter",
		in:     BadIn{Value: 1},
		target: ErrInvalidRule,
	},
	{
		name:   "unsupported field type",
		in:     BadType{Value: 1.5},
		target: ErrUnsupportedType,
	},
	{
		name:   "rule without parameter",
		in:     BadSyntax{Value: 1},
		target: ErrInvalidRule,
	},
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func requireProgramError(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("Validate() error = %v, want %v", err, target)
	}

	var validationErrors ValidationErrors
	if errors.As(err, &validationErrors) {
		t.Fatalf("Validate() returned validation errors %v, want program error", validationErrors)
	}
}

func requireValidationErrors(t *testing.T, err error, expected []fieldError) {
	t.Helper()

	var validationErrors ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("Validate() error = %v, want ValidationErrors", err)
	}

	if len(validationErrors) != len(expected) {
		t.Fatalf("got %d validation errors (%v), want %d", len(validationErrors), validationErrors, len(expected))
	}

	for i, want := range expected {
		if validationErrors[i].Field != want.field {
			t.Errorf("error %d field = %q, want %q", i, validationErrors[i].Field, want.field)
		}
		if !errors.Is(validationErrors[i].Err, want.target) {
			t.Errorf("error %d = %v, want errors.Is(_, %v)", i, validationErrors[i].Err, want.target)
		}
	}
}

func validUser() User {
	return User{
		ID:     strings.Repeat("a", 36),
		Name:   "Alice",
		Age:    25,
		Email:  "alice@example.com",
		Role:   "admin",
		Phones: []string{"12345678901", "10987654321"},
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	for _, tt := range validationTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tt.in)
			switch {
			case tt.target != nil:
				requireProgramError(t, err, tt.target)
			case tt.expected != nil:
				requireValidationErrors(t, err, tt.expected)
			default:
				requireNoError(t, err)
			}
		})
	}
}

func TestValidationErrorsErrorMessage(t *testing.T) {
	t.Parallel()

	errs := ValidationErrors{
		{Field: "Age", Err: ErrMin},
		{Field: "Name", Err: ErrLen},
	}

	message := errs.Error()
	if !strings.Contains(message, "Age") || !strings.Contains(message, "Name") {
		t.Fatalf("Error() = %q, want it to mention both fields", message)
	}
}

func ptr[T any](value T) *T {
	return &value
}
