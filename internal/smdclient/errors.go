package smdclient

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnmarshal = errors.New("cannot unmarshal JSON")
	ErrEmptyID   = errors.New("empty id")
)

// ErrSMDResponse contains the HTTP response of a REST API request to SMD.
type ErrSMDResponse struct {
	HTTPResponse *http.Response
}

func (esr ErrSMDResponse) Error() string {
	return fmt.Sprintf("SMD response returned %s", esr.HTTPResponse.Status)
}
