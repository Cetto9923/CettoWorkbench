package schedule

// TaskMutationError is a deterministic validation failure for a task mutation.
// Code is mapped by the HTTP handler; the Service remains transport agnostic.
type TaskMutationError struct {
	Code    string
	Message string
}

func (e *TaskMutationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

const (
	taskMutationNotFound  = "not_found"
	taskMutationForbidden = "forbidden"
	taskMutationConflict  = "conflict"
)
