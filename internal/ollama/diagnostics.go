package ollama

import "net/url"

// DiagnosticURL returns the endpoint without credentials, query parameters, or
// fragments. It does not modify u.
func DiagnosticURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	safe := u.Clone()
	safe.User = nil
	safe.RawQuery = ""
	safe.ForceQuery = false
	safe.Fragment = ""
	safe.RawFragment = ""
	return safe.String()
}

// HTTP errors can include the complete request URL. Keep their causes
// inspectable without printing credentials or query values in diagnostics.
type diagnosticRequestError struct{ cause error }

func (diagnosticRequestError) Error() string { return "HTTP request failed" }

func (e diagnosticRequestError) Unwrap() error { return e.cause }
