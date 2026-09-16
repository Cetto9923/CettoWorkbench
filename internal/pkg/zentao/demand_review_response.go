package zentao

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// extractLeadingJSON returns the first complete JSON value in body when
// trailing non-JSON (e.g. HTML from ZenTao display fallthrough) is present.
// If the body has no valid leading JSON, the original body is returned unchanged.
func extractLeadingJSON(body []byte) []byte {
	dec := json.NewDecoder(bytes.NewReader(body))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return body
	}
	return []byte(raw)
}

// validateDemandReviewResponse rejects controller failures even when HTTP is 200.
// Older deployed entries encode their JSON object twice.
// ZenTao may append HTML after a successful JSON send (display fallthrough);
// we tolerate that by decoding only the leading JSON value.
func validateDemandReviewResponse(body []byte, status int) error {
	if status < 200 || status >= 300 {
		return parseZentaoAPIError(body, status)
	}
	body = extractLeadingJSON(body)
	var encoded string
	if json.Unmarshal(body, &encoded) == nil {
		body = []byte(encoded)
	}
	var result struct {
		Success *bool           `json:"success"`
		Status  string          `json:"status"`
		Result  string          `json:"result"`
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &result) != nil {
		return fmt.Errorf("%w: invalid demand action response", ErrZentaoAPIError)
	}
	if (result.Success != nil && !*result.Success) || result.Status == "fail" || result.Status == "error" || result.Result == "fail" || result.Result == "error" || (len(result.Error) > 0 && string(result.Error) != "null") {
		return parseZentaoAPIError(body, status)
	}
	if (result.Success != nil && *result.Success) || result.Status == "success" || result.Result == "success" || (result.Status == "" && result.Result == "" && result.Message != "") {
		return nil
	}
	return fmt.Errorf("%w: missing demand action result", ErrZentaoAPIError)
}
