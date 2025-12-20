package models

type BusinessError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *BusinessError) Error() string { return e.Message }

func BizError(code, msg string) *BusinessError { return &BusinessError{Code: code, Message: msg} }
