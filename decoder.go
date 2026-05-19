package godantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

func decodeError(err error) error {
	switch e := err.(type) {
	case *json.UnmarshalTypeError:
		return &Error{
			ErrType: "TYPE_MISMATCH_ERR",
			Path:    e.Field,
			Message: fmt.Sprintf("The field <%s> was given an invalid type, the expected type is `%s`", e.Field, e.Type.String()),
		}

	case *json.SyntaxError:
		return &Error{
			ErrType: "SYNTAX_ERR",
			Path:    e.Error(),
			Message: e.Error(),
		}
	case *time.ParseError:
		return &Error{
			ErrType: "INVALID_TIME_ERR",
			Path:    "",
			Message: fmt.Sprintf("Invalid time <%s>, expected format `%s`", e.Value, e.Layout),
		}
	default:
		return nil
	}
}

func decodeJSON(jsonData []byte, obj interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	err := decoder.Decode(obj)
	if err != nil {
		return decodeError(err)
	}
	return nil
}

func (g *Validate) BindJSON(jsonData []byte, obj any) error {
	err := decodeJSON(jsonData, obj)
	if err != nil {
		return err
	}
	var reqDataMap map[string]any
	err = json.Unmarshal(jsonData, &reqDataMap)
	if err != nil {
		return &Error{
			ErrType: "INVALID_JSON_ERR",
			Path:    "",
			Message: "The given data is not a valid JSON",
		}
	}
	if len(reqDataMap) == 0 {
		return &Error{
			ErrType: "EMPTY_JSON_ERR",
			Path:    "",
			Message: "The given json data is empty",
		}
	}

	err = g.InspectStruct(obj)
	if err != nil {
		return err
	}

	err = g.CheckTypeCompatibility(reqDataMap, buildRefData(obj))
	if err != nil {
		return err
	}

	return nil
}

func (e *Error) Error() string {
	return e.Message
}

type Error struct {
	ErrType string
	Message string
	Path    string
}

// CustomErr is a type alias for Error, kept for API compatibility with ValidationPlugin implementations.
type CustomErr = Error
