package zentao

import (
	"encoding/json"
	"fmt"
)

// validateDemandReviewResponse rejects controller failures even when HTTP is 200.
// Older deployed entries encode their JSON object twice.
func validateDemandReviewResponse(body []byte, status int) error {
	if status < 200 || status >= 300 {
		return parseZentaoAPIError(body, status)
	}
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
