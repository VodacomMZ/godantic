# Godantic Roadmap

This document tracks planned features and improvements that are not yet implemented.

---

## Planned

### Multi-error accumulation

**Status:** Not started

Currently `BindJSON` / `InspectStruct` return on the first validation failure. Users who submit a form with multiple invalid fields must fix them one at a time, round-tripping for each error.

The goal is to collect all validation errors in a single pass and return them together:

```go
errs, err := v.BindJSONAll(data, &body)
// errs is []ValidationError, one entry per failing field
```

**Design notes:**

- Thread an `[]error` accumulator through `checkStruct` / `checkField` / `checkList` instead of returning on first failure.
- Introduce a `ValidationErrors` type (`[]Error`) that implements the `error` interface, so callers that only check `err != nil` keep working.
- The existing `BindJSON` / `InspectStruct` API stays unchanged — `BindJSONAll` is the new opt-in entry point.
- Slice element errors should carry their index in the path (`items[1].name`), which is already in place.

**Motivation:** Most validation libraries (Python's Pydantic, Java's Bean Validation, JS's Zod) return all errors at once. This is the single change that would most improve the user-facing experience.

---

## Completed (recent)

- Indexed slice element error paths (`items[0].name` not `items.name`)
- `when` conditional tag works on any string field, not only `enum`-tagged fields
- `buildRefData` recurses into pointer struct fields for unknown-field detection
- Deep-nesting path accumulation fixed (3+ levels now produce correct paths)
- `validateWithCustomTag` dereferences pointer values before passing to validators
- `Error.Error()` no longer mutates the receiver
- `formatValidation` compiles regex only once per call
- `ClearCustomValidators()` for test isolation
- Whitespace-only required plain strings now rejected (consistent with `*string` behaviour)
