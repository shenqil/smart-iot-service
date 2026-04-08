package errx

import (
	"fmt"
)

// Error 结构化错误类型
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func New(code int, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
