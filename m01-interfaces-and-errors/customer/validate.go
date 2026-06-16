package customer

// M01 Exercise 3.2 — a validation error that carries data.
//
// Build here:
//
//   - `ValidationError` struct with `Field string` and `Msg string`, and an
//     Error() method (pointer receiver) like:
//         "validation: <field>: <msg>"
//     Returning a typed error lets callers extract the offending field with
//     errors.As(err, &ve).
//
//   - `(c Customer) Validate() error` — note the return type is the `error`
//     interface, NOT *ValidationError (avoids the typed-nil trap and lets you
//     combine failures). Use errors.Join to combine per-field checks:
//         - email must contain "@"      -> &ValidationError{Field:"email", ...}
//         - name must not be blank/space -> &ValidationError{Field:"name", ...}
//     A passing field contributes nil; errors.Join drops nils for you.
//
// TODO(3.2): implement ValidationError and (Customer).Validate.
