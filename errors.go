package appkit

import (
    "errors"
)

// general error response for returning errors in json format...
type errorResponse struct {
    Error string `json:"error"`
}

func ErrorResponseErr(err error) errorResponse {
    return errorResponse{Error: err.Error()}
}

func ErrorResponseMsg(s string) errorResponse {
    return errorResponse{Error: s}
}

var ErrConnectionInfoIncomplete = errors.New("Connection information incomplete")
