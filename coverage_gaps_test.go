package godantic

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────
// decodeError: all branches
// ─────────────────────────────────────────────────────────────

func TestDecodeError_UnmarshalTypeError(t *testing.T) {
	type S struct {
		Age int `json:"age"`
	}
	var s S
	g := &Validate{}
	err := g.BindJSON([]byte(`{"age": "not-a-number"}`), &s)
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok, "expected *Error")
	assert.Equal(t, "TYPE_MISMATCH_ERR", e.ErrType)
}

func TestDecodeError_SyntaxError(t *testing.T) {
	type S struct{ Name string `json:"name"` }
	var s S
	g := &Validate{}
	err := g.BindJSON([]byte(`{bad json}`), &s)
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok, "expected *Error")
	assert.Equal(t, "SYNTAX_ERR", e.ErrType)
}

func TestDecodeError_TimeParseError(t *testing.T) {
	parseErr := &time.ParseError{Layout: time.RFC3339, Value: "bad-time"}
	err := decodeError(parseErr)
	assert.NotNil(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TIME_ERR", e.ErrType)
}

func TestDecodeError_Default(t *testing.T) {
	err := decodeError(fmt.Errorf("some unrecognized error type"))
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// CustomErr.Error()
// ─────────────────────────────────────────────────────────────

func TestCustomErrError(t *testing.T) {
	e := &CustomErr{ErrType: "TEST_ERR", Message: "custom error message", Path: "some.path"}
	assert.Equal(t, "custom error message", e.Error())
	assert.Equal(t, "custom error message", e.Error()) // idempotent
}

// ─────────────────────────────────────────────────────────────
// checkTime: zero and valid time.Time (direct call)
// ─────────────────────────────────────────────────────────────

func TestCheckTime_ZeroTime(t *testing.T) {
	g := &Validate{}
	err := g.checkTime(reflect.ValueOf(time.Time{}), "created_at")
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TIME_ERR", e.ErrType)
	assert.Equal(t, "created_at", e.Path)
}

func TestCheckTime_ValidTime(t *testing.T) {
	g := &Validate{}
	err := g.checkTime(reflect.ValueOf(time.Now()), "created_at")
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// inspect: *time.Time field — isTime now checked before isStruct
// ─────────────────────────────────────────────────────────────

func TestInspect_PtrTime_NonZeroPasses(t *testing.T) {
	type WithPtrTime struct {
		Name      *string    `json:"name"`
		CreatedAt *time.Time `json:"created_at"`
	}
	now := time.Now()
	name := "test"
	g := &Validate{}
	err := g.InspectStruct(WithPtrTime{Name: &name, CreatedAt: &now})
	assert.Nil(t, err)
}

func TestInspect_PtrTime_ZeroFails(t *testing.T) {
	type WithPtrTime struct {
		Name      *string    `json:"name"`
		CreatedAt *time.Time `json:"created_at"`
	}
	zero := time.Time{}
	name := "test"
	g := &Validate{}
	err := g.InspectStruct(WithPtrTime{Name: &name, CreatedAt: &zero})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TIME_ERR", e.ErrType)
}

func TestInspect_SliceOfTime_ZeroFails(t *testing.T) {
	g := &Validate{}
	times := []time.Time{time.Now(), time.Time{}}
	err := g.InspectStruct(times)
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TIME_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// checkStruct: optional time.Time field (no binding:"required") is
// skipped even when zero; required time.Time must be non-zero.
// ─────────────────────────────────────────────────────────────

func TestCheckStruct_OptionalTimeFieldAllowsZero(t *testing.T) {
	type WithTime struct {
		Name      string    `json:"name" binding:"required"`
		CreatedAt time.Time `json:"created_at"`
	}
	g := &Validate{}
	err := g.InspectStruct(WithTime{Name: "Alice"})
	assert.Nil(t, err)
}

func TestCheckStruct_RequiredTimeFieldRejectsZero(t *testing.T) {
	type WithTime struct {
		Name      string    `json:"name" binding:"required"`
		CreatedAt time.Time `json:"created_at" binding:"required"`
	}
	g := &Validate{}
	err := g.InspectStruct(WithTime{Name: "Alice"}) // CreatedAt is zero
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TIME_ERR", e.ErrType)
}

func TestCheckStruct_RequiredTimeFieldAcceptsNonZero(t *testing.T) {
	type WithTime struct {
		Name      string    `json:"name" binding:"required"`
		CreatedAt time.Time `json:"created_at" binding:"required"`
	}
	g := &Validate{}
	err := g.InspectStruct(WithTime{Name: "Alice", CreatedAt: time.Now()})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkList: IgnoreMinLen=true allows an empty slice
// ─────────────────────────────────────────────────────────────

func TestCheckList_IgnoreMinLen_EmptySliceAllowed(t *testing.T) {
	type Item struct {
		Name string `json:"name" binding:"required"`
	}
	g := &Validate{IgnoreMinLen: true}
	var empty []Item
	err := g.InspectStruct(empty)
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkList: int elements hit the inspect default branch
// ─────────────────────────────────────────────────────────────

func TestCheckList_IntElementsHitDefault(t *testing.T) {
	g := &Validate{}
	err := g.InspectStruct([]int{1, 2, 3})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkList: ValidationPlugin error via pointer receiver
//
// When the plugin is implemented with a pointer receiver, the
// value-based path through inspect/validateInterfaceHooks does
// not detect it — only the explicit resolveInterface call in
// checkList catches it.
// ─────────────────────────────────────────────────────────────

type ptrPlugin struct{ fail bool }

func (p *ptrPlugin) Validate() *CustomErr {
	if p.fail {
		return &CustomErr{ErrType: "PTR_PLUGIN_ERR", Message: "pointer plugin failed"}
	}
	return nil
}

func TestCheckList_PtrPluginValidationError(t *testing.T) {
	g := &Validate{}
	items := []*ptrPlugin{{fail: true}}
	err := g.InspectStruct(items)
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "PTR_PLUGIN_ERR", e.ErrType)
}

func TestCheckList_PtrPluginValidationSuccess(t *testing.T) {
	g := &Validate{}
	items := []*ptrPlugin{{fail: false}}
	err := g.InspectStruct(items)
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkList: DynamicFieldsValidator error via pointer receiver
// ─────────────────────────────────────────────────────────────

type ptrDynamic struct {
	val   any
	vtype string
	attr  string
}

func (p *ptrDynamic) GetValue() any        { return p.val }
func (p *ptrDynamic) GetValueType() string { return p.vtype }
func (p *ptrDynamic) GetAttribute() string { return p.attr }

func TestCheckList_PtrDynamicValidationError(t *testing.T) {
	g := &Validate{}
	items := []*ptrDynamic{{val: "not-a-number", vtype: "integer", attr: "age"}}
	err := g.InspectStruct(items)
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_VALUE_TYPE_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// strEnums: nil pointer optional enum field — should not error
// ─────────────────────────────────────────────────────────────

func TestStrEnums_NilPointerOptionalEnum(t *testing.T) {
	type OptionalRole struct {
		Role *string `json:"role" enum:"admin,user"`
	}
	g := &Validate{}
	err := g.InspectStruct(OptionalRole{Role: nil})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkField: unexported field is silently skipped
// ─────────────────────────────────────────────────────────────

func TestCheckField_UnexportedFieldSkipped(t *testing.T) {
	type withUnexported struct {
		Name   string `json:"name" binding:"required"`
		secret string //nolint
	}
	g := &Validate{}
	err := g.InspectStruct(withUnexported{Name: "Alice"})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkField: ValidationPlugin on a named non-struct type
//
// For fields whose Kind is not Ptr or Struct, checkField skips
// inspect/checkStruct, so the only plugin check is the explicit
// resolveInterface block after the switch.
// ─────────────────────────────────────────────────────────────

type namedInt int

func (n namedInt) Validate() *CustomErr {
	if n < 0 {
		return &CustomErr{ErrType: "NEGATIVE_ERR", Message: "must be non-negative"}
	}
	return nil
}

func TestCheckField_ValidationPlugin_NamedNonStructType(t *testing.T) {
	type withNamedInt struct {
		Value namedInt `json:"value"`
	}
	g := &Validate{}

	err := g.InspectStruct(withNamedInt{Value: 5})
	assert.Nil(t, err)

	err = g.InspectStruct(withNamedInt{Value: -1})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "NEGATIVE_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// validateField: Object case with a non-map request value
// ─────────────────────────────────────────────────────────────

func TestValidateField_ObjectWithNonMapValue(t *testing.T) {
	g := &Validate{}
	reqData := map[string]any{"meta": 42}
	refData := map[string]any{"meta": Object{}}
	err := g.CheckTypeCompatibility(reqData, refData)
	assert.Error(t, err)
}

// ─────────────────────────────────────────────────────────────
// validateList: request value is not a list
// ─────────────────────────────────────────────────────────────

func TestValidateList_ReqValueNotAList(t *testing.T) {
	g := &Validate{}
	reqData := map[string]any{"items": "not-a-list"}
	refData := map[string]any{"items": []any{"template"}}
	err := g.CheckTypeCompatibility(reqData, refData)
	assert.Error(t, err)
}

func TestValidateList_RefListNotAList_DirectCall(t *testing.T) {
	g := &Validate{}
	err := g.validateList("not-a-list", []any{"a"}, "path")
	assert.Error(t, err)
}

// ─────────────────────────────────────────────────────────────
// validateDynamicFields: unknown value type hits the default case
// ─────────────────────────────────────────────────────────────

func TestValidateDynamicFields_UnknownType(t *testing.T) {
	err := validateDynamicFields("anything", "field", "unknowntype", "root")
	assert.NotNil(t, err)
	assert.Equal(t, "INVALID_VALUE_TYPE_ERR", err.ErrType)
}

// ─────────────────────────────────────────────────────────────
// buildRefData: json:"-", map field, slice of primitive, ptr input,
//               and field with no json tag
// ─────────────────────────────────────────────────────────────

func TestBuildRefData_JsonDashSkipped(t *testing.T) {
	type S struct {
		Name   string `json:"name"`
		Hidden string `json:"-"`
	}
	result := buildRefData(S{Name: "Alice"})
	_, hasHidden := result["hidden"]
	_, hasDash := result["-"]
	assert.False(t, hasHidden, "json:\"-\" field must be omitted")
	assert.False(t, hasDash, "json:\"-\" field must be omitted")
	_, hasName := result["name"]
	assert.True(t, hasName)
}

func TestBuildRefData_MapField(t *testing.T) {
	type S struct {
		Name string            `json:"name"`
		Meta map[string]string `json:"meta"`
	}
	result := buildRefData(S{})
	v, ok := result["meta"]
	assert.True(t, ok)
	_, isMap := v.(map[string]any)
	assert.True(t, isMap)
}

func TestBuildRefData_SliceOfPrimitive(t *testing.T) {
	type S struct {
		Tags []string `json:"tags"`
	}
	result := buildRefData(S{})
	v, ok := result["tags"]
	assert.True(t, ok)
	slice, isSlice := v.([]any)
	assert.True(t, isSlice)
	assert.Empty(t, slice)
}

func TestBuildRefData_NoJsonTag(t *testing.T) {
	type S struct {
		NoTag string
	}
	result := buildRefData(S{})
	_, hasField := result["NoTag"]
	assert.True(t, hasField, "field with no json tag should use the field Name")
}

func TestBuildRefData_PtrInput(t *testing.T) {
	type S struct {
		Name string `json:"name"`
	}
	result := buildRefData(&S{Name: "test"})
	_, hasName := result["name"]
	assert.True(t, hasName)
}

// ─────────────────────────────────────────────────────────────
// parseCondition: part without an operator is ignored
// ─────────────────────────────────────────────────────────────

func TestParseCondition_PartWithoutOperator(t *testing.T) {
	conditions, bindings := parseCondition("no-operator;binding=required")
	assert.Empty(t, conditions, "part without operator must be ignored")
	assert.Equal(t, "required", bindings["binding"])
}

func TestParseCondition_EmptyTag(t *testing.T) {
	conditions, bindings := parseCondition("")
	assert.Empty(t, conditions)
	assert.Empty(t, bindings)
}

// ─────────────────────────────────────────────────────────────
// validateCondition: binding type other than "required" is a no-op
// ─────────────────────────────────────────────────────────────

func TestValidateCondition_UnknownBindingType(t *testing.T) {
	type Inner struct {
		Type *string `json:"type" enum:"a,b"`
	}
	type WithUnknownBinding struct {
		Inner Inner  `json:"inner"`
		Field string `json:"field" when:"inner.type=a;binding=unknown"`
	}
	g := &Validate{}
	typeVal := "a"
	err := g.InspectStruct(WithUnknownBinding{Inner: Inner{Type: &typeVal}, Field: "hello"})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// formatValidation: nil pointer with a format tag skips cleanly
// ─────────────────────────────────────────────────────────────

func TestFormatValidation_NilPointerSkipped(t *testing.T) {
	type S struct {
		Email *string `json:"email" format:"email"`
	}
	g := &Validate{}
	err := g.InspectStruct(S{Email: nil})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// regexPattern: nil pointer with a regex tag skips cleanly
// ─────────────────────────────────────────────────────────────

func TestRegexPattern_NilPointerSkipped(t *testing.T) {
	type S struct {
		Code *string `json:"code" regex:"^[A-Z]+$"`
	}
	g := &Validate{}
	err := g.InspectStruct(S{Code: nil})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// matchRegexPattern: invalid (un-compilable) regex returns an error
// ─────────────────────────────────────────────────────────────

func TestMatchRegexPattern_InvalidRegex(t *testing.T) {
	err := matchRegexPattern("[invalid(regex", "value", reflect.StructField{}, "path")
	assert.Error(t, err)
}

// ─────────────────────────────────────────────────────────────
// getFormatRegex: unknown format tag returns an empty string
// ─────────────────────────────────────────────────────────────

func TestGetFormatRegex_UnknownFormat(t *testing.T) {
	result := getFormatRegex("unknown_format_xyz")
	assert.Equal(t, "", result)
}

// ─────────────────────────────────────────────────────────────
// checkNumericConstraints: allow_inf_nan:"true" permits NaN / Inf
// ─────────────────────────────────────────────────────────────

func TestCheckNumericConstraints_AllowInfNaN(t *testing.T) {
	type S struct {
		Value *float64 `json:"value" allow_inf_nan:"true"`
	}
	g := &Validate{}

	infVal := math.Inf(1)
	err := g.InspectStruct(S{Value: &infVal})
	assert.Nil(t, err)
}

func TestCheckNumericConstraints_RejectInfByDefault(t *testing.T) {
	type S struct {
		Value *float64 `json:"value"`
	}
	g := &Validate{}

	infVal := math.Inf(1)
	err := g.InspectStruct(S{Value: &infVal})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_FLOAT_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// validateWithCustomTag: pointer-type field dereferences the type;
//                        empty segment in tag list is skipped;
//                        no registered validator is a no-op
// ─────────────────────────────────────────────────────────────

func TestValidateWithCustomTag_PtrFieldTypeDerefsForLookup(t *testing.T) {
	// validateWithCustomTag dereferences the field *type* (not the value)
	// to look up the validator. This test drives the t.Kind()==reflect.Ptr branch.
	RegisterCustom[string]("ptr_deref_check", func(val string, path string) *Error {
		return nil
	})

	type S struct {
		Name *string `json:"name" validate:"ptr_deref_check"`
	}
	g := &Validate{}

	// Call directly: field type is *string, value is already a plain string.
	// The type is dereffed to string → validator is found and succeeds.
	typ := reflect.TypeOf(S{})
	nameField := typ.Field(0)
	err := g.validateWithCustomTag("Alice", nameField, "name")
	assert.Nil(t, err)
}

func TestValidateWithCustomTag_EmptySegmentInTag(t *testing.T) {
	type S struct {
		Name string `json:"name" validate:",nonempty"`
	}
	RegisterCustom[string]("nonempty", func(val string, path string) *Error {
		if val == "" {
			return &Error{ErrType: "EMPTY_ERR", Message: "must not be empty"}
		}
		return nil
	})
	g := &Validate{}
	err := g.InspectStruct(S{Name: "Alice"})
	assert.Nil(t, err)
}

func TestValidateWithCustomTag_NoRegisteredValidator(t *testing.T) {
	type S struct {
		Count int `json:"count" validate:"not_registered_for_int"`
	}
	g := &Validate{}
	err := g.InspectStruct(S{Count: 5})
	assert.Nil(t, err)
}

func TestValidateWithCustomTag_TypeMismatchInWrapper(t *testing.T) {
	// RegisterCustom wraps fn with a type assertion; passing the wrong
	// value type triggers the INVALID_TYPE_ERR path inside the wrapper.
	RegisterCustom[string]("type_mismatch_test", func(val string, path string) *Error {
		return nil
	})
	type S struct {
		Field string `json:"field" validate:"type_mismatch_test"`
	}
	g := &Validate{}
	typ := reflect.TypeOf(S{})
	field := typ.Field(0) // string field
	// Pass an int instead of a string — the wrapper assertion int→string fails.
	err := g.validateWithCustomTag(42, field, "field")
	assert.NotNil(t, err)
	assert.Equal(t, "INVALID_TYPE_ERR", err.ErrType)
}

// ─────────────────────────────────────────────────────────────
// strEnums: invalid enum value triggers INVALID_ENUM_ERR
// ─────────────────────────────────────────────────────────────

func TestStrEnums_InvalidValue(t *testing.T) {
	type WithEnum struct {
		Role string `json:"role" enum:"admin,user"`
	}
	g := &Validate{}
	err := g.InspectStruct(WithEnum{Role: "superuser"})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_ENUM_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// validateField: refType is a map but reqValue is not a map
// ─────────────────────────────────────────────────────────────

func TestValidateField_MapTypeWithNonMapValue(t *testing.T) {
	g := &Validate{}
	reqData := map[string]any{"nested": "not-a-map"}
	refData := map[string]any{"nested": map[string]any{"key": ""}}
	err := g.CheckTypeCompatibility(reqData, refData)
	assert.Error(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkField: DynamicFieldsValidator on a named non-struct type
//
// Named string type with a value-receiver DynamicFieldsValidator
// is not caught by validateInterfaceHooks (which operates on the
// parent struct); only the explicit block at the end of checkField
// detects it.
// ─────────────────────────────────────────────────────────────

type dynStr string

func (d dynStr) GetValue() any        { return 42 }      // int value but claims string type
func (d dynStr) GetValueType() string { return "string" }
func (d dynStr) GetAttribute() string { return "field" }

func TestCheckField_DynamicFieldsValidator_NamedNonStructType(t *testing.T) {
	type withDynStr struct {
		Field dynStr `json:"field"`
	}
	g := &Validate{}
	err := g.InspectStruct(withDynStr{Field: "something"})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_VALUE_TYPE_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// buildRefData: Object field and slice-of-struct field
// ─────────────────────────────────────────────────────────────

func TestBuildRefData_ObjectField(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type S struct {
		Meta  Object `json:"meta"`
		Items []Item `json:"items"`
	}
	result := buildRefData(S{})

	_, isObject := result["meta"].(Object)
	assert.True(t, isObject, "Object field should be stored as Object{}")

	items, isSlice := result["items"].([]any)
	assert.True(t, isSlice)
	assert.NotEmpty(t, items, "slice of struct should include one template element")
}

// ─────────────────────────────────────────────────────────────
// checkNumericConstraints: uint field
// ─────────────────────────────────────────────────────────────

func TestCheckNumericConstraints_UintField(t *testing.T) {
	type S struct {
		Count uint `json:"count" gt:"0"`
	}
	g := &Validate{}

	err := g.InspectStruct(S{Count: 5})
	assert.Nil(t, err)

	err = g.InspectStruct(S{Count: 0})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "GREATER_THAN_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// parseCondition: empty segment (blank between semicolons)
// ─────────────────────────────────────────────────────────────

func TestParseCondition_EmptySegment(t *testing.T) {
	// Leading semicolon produces an empty first segment that must be skipped.
	conditions, bindings := parseCondition(";context.type=org;binding=required")
	assert.Equal(t, "org", conditions["context.type"])
	assert.Equal(t, "required", bindings["binding"])
}

// ─────────────────────────────────────────────────────────────
// validateList: empty ref list skips item validation
// ─────────────────────────────────────────────────────────────

func TestValidateList_EmptyRefListSkipsValidation(t *testing.T) {
	g := &Validate{}
	// ref list is empty → validateList returns nil immediately without
	// checking any request items (the len(refList)==0 branch).
	reqData := map[string]any{"items": []any{"a", "b", "c"}}
	refData := map[string]any{"items": []any{}}
	err := g.CheckTypeCompatibility(reqData, refData)
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkNumericConstraints: NaN is rejected by default
// ─────────────────────────────────────────────────────────────

func TestCheckNumericConstraints_NaNRejected(t *testing.T) {
	type S struct {
		Value *float64 `json:"value"`
	}
	g := &Validate{}
	nan := math.NaN()
	err := g.InspectStruct(S{Value: &nan})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_FLOAT_ERR", e.ErrType)
}

// ─────────────────────────────────────────────────────────────
// BindJSON: unknown fields are now rejected (CheckTypeCompatibility
// args were previously swapped so this check never fired).
// ─────────────────────────────────────────────────────────────

func TestBindJSON_RejectsUnknownField(t *testing.T) {
	type Body struct {
		Name string `json:"name" binding:"required"`
	}
	g := &Validate{}
	err := g.BindJSON([]byte(`{"name":"alice","extra":"x"}`), &Body{})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_FIELD_ERR", e.ErrType)
}

func TestBindJSON_AllowsUnknownFieldWhenFlagSet(t *testing.T) {
	type Body struct {
		Name string `json:"name" binding:"required"`
	}
	g := &Validate{AllowUnknownFields: true}
	err := g.BindJSON([]byte(`{"name":"alice","extra":"x"}`), &Body{})
	assert.Nil(t, err)
}

// ─────────────────────────────────────────────────────────────
// checkField: plain (non-pointer) slice struct fields now have
// their elements recursively inspected.
// ─────────────────────────────────────────────────────────────

func TestCheckField_PlainSliceElementsValidated(t *testing.T) {
	type Item struct {
		Val string `json:"val" binding:"required"`
	}
	type Body struct {
		Items []Item `json:"items"`
	}
	g := &Validate{}
	// Item with an empty Val — should fail required check on the element.
	err := g.InspectStruct(Body{Items: []Item{{Val: ""}}})
	assert.Error(t, err)
	e, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, "REQUIRED_FIELD_ERR", e.ErrType)
}

func TestCheckField_PlainSliceValidElementsPass(t *testing.T) {
	type Item struct {
		Val string `json:"val" binding:"required"`
	}
	type Body struct {
		Items []Item `json:"items"`
	}
	g := &Validate{}
	err := g.InspectStruct(Body{Items: []Item{{Val: "hello"}}})
	assert.Nil(t, err)
}

// buildRefData must represent time.Time fields as "" (string) not as a
// map of its unexported internals, so that CheckTypeCompatibility does
// not produce false "expected map" errors for time fields.
func TestBuildRefData_TimeFieldIsString(t *testing.T) {
	type S struct {
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}
	result := buildRefData(S{})
	v, ok := result["created_at"]
	assert.True(t, ok)
	_, isString := v.(string)
	assert.True(t, isString, "time.Time field must appear as string in schema")
}
