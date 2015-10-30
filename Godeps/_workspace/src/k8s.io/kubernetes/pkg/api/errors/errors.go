/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/errors"
	"k8s.io/kubernetes/pkg/util/fielderrors"
)

// HTTP Status codes not in the golang http package.
const (
	StatusUnprocessableEntity = 422
	StatusTooManyRequests     = 429
	// HTTP recommendations are for servers to define 5xx error codes
	// for scenarios not covered by behavior. In this case, ServerTimeout
	// is an indication that a transient server error has occurred and the
	// client *should* retry, with an optional Retry-After header to specify
	// the back off window.
	StatusServerTimeout = 504
)

// StatusError is an error intended for consumption by a REST API server; it can also be
// reconstructed by clients from a REST response. Public to allow easy type switches.
type StatusError struct {
	ErrStatus unversioned.Status
}

var _ error = &StatusError{}

// Error implements the Error interface.
func (e *StatusError) Error() string {
	return e.ErrStatus.Message
}

// Status allows access to e's status without having to know the detailed workings
// of StatusError. Used by pkg/apiserver.
func (e *StatusError) Status() unversioned.Status {
	return e.ErrStatus
}

// DebugError reports extended info about the error to debug output.
func (e *StatusError) DebugError() (string, []interface{}) {
	if out, err := json.MarshalIndent(e.ErrStatus, "", "  "); err == nil {
		return "server response object: %s", []interface{}{string(out)}
	}
	return "server response object: %#v", []interface{}{e.ErrStatus}
}

// UnexpectedObjectError can be returned by FromObject if it's passed a non-status object.
type UnexpectedObjectError struct {
	Object runtime.Object
}

// Error returns an error message describing 'u'.
func (u *UnexpectedObjectError) Error() string {
	return fmt.Sprintf("unexpected object: %v", u.Object)
}

// FromObject generates an StatusError from an unversioned.Status, if that is the type of obj; otherwise,
// returns an UnexpecteObjectError.
func FromObject(obj runtime.Object) error {
	switch t := obj.(type) {
	case *unversioned.Status:
		return &StatusError{*t}
	}
	return &UnexpectedObjectError{obj}
}

// NewNotFound returns a new error which indicates that the resource of the kind and the name was not found.
func NewNotFound(kind, name string) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusNotFound,
		Reason: unversioned.StatusReasonNotFound,
		Details: &unversioned.StatusDetails{
			Kind: kind,
			Name: name,
		},
		Message: fmt.Sprintf("%s %q not found", kind, name),
	}}
}

// NewAlreadyExists returns an error indicating the item requested exists by that identifier.
func NewAlreadyExists(kind, name string) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusConflict,
		Reason: unversioned.StatusReasonAlreadyExists,
		Details: &unversioned.StatusDetails{
			Kind: kind,
			Name: name,
		},
		Message: fmt.Sprintf("%s %q already exists", kind, name),
	}}
}

// NewUnauthorized returns an error indicating the client is not authorized to perform the requested
// action.
func NewUnauthorized(reason string) error {
	message := reason
	if len(message) == 0 {
		message = "not authorized"
	}
	return &StatusError{unversioned.Status{
		Status:  unversioned.StatusFailure,
		Code:    http.StatusUnauthorized,
		Reason:  unversioned.StatusReasonUnauthorized,
		Message: message,
	}}
}

// NewForbidden returns an error indicating the requested action was forbidden
func NewForbidden(kind, name string, err error) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusForbidden,
		Reason: unversioned.StatusReasonForbidden,
		Details: &unversioned.StatusDetails{
			Kind: kind,
			Name: name,
		},
		Message: fmt.Sprintf("%s %q is forbidden: %v", kind, name, err),
	}}
}

// NewConflict returns an error indicating the item can't be updated as provided.
func NewConflict(kind, name string, err error) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusConflict,
		Reason: unversioned.StatusReasonConflict,
		Details: &unversioned.StatusDetails{
			Kind: kind,
			Name: name,
		},
		Message: fmt.Sprintf("%s %q cannot be updated: %v", kind, name, err),
	}}
}

// NewInvalid returns an error indicating the item is invalid and cannot be processed.
func NewInvalid(kind, name string, errs fielderrors.ValidationErrorList) error {
	causes := make([]unversioned.StatusCause, 0, len(errs))
	for i := range errs {
		if err, ok := errs[i].(*fielderrors.ValidationError); ok {
			causes = append(causes, unversioned.StatusCause{
				Type:    unversioned.CauseType(err.Type),
				Message: err.ErrorBody(),
				Field:   err.Field,
			})
		}
	}
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   StatusUnprocessableEntity, // RFC 4918: StatusUnprocessableEntity
		Reason: unversioned.StatusReasonInvalid,
		Details: &unversioned.StatusDetails{
			Kind:   kind,
			Name:   name,
			Causes: causes,
		},
		Message: fmt.Sprintf("%s %q is invalid: %v", kind, name, errors.NewAggregate(errs)),
	}}
}

// NewBadRequest creates an error that indicates that the request is invalid and can not be processed.
func NewBadRequest(reason string) error {
	return &StatusError{unversioned.Status{
		Status:  unversioned.StatusFailure,
		Code:    http.StatusBadRequest,
		Reason:  unversioned.StatusReasonBadRequest,
		Message: reason,
	}}
}

// NewServiceUnavailable creates an error that indicates that the requested service is unavailable.
func NewServiceUnavailable(reason string) error {
	return &StatusError{unversioned.Status{
		Status:  unversioned.StatusFailure,
		Code:    http.StatusServiceUnavailable,
		Reason:  unversioned.StatusReasonServiceUnavailable,
		Message: reason,
	}}
}

// NewMethodNotSupported returns an error indicating the requested action is not supported on this kind.
func NewMethodNotSupported(kind, action string) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusMethodNotAllowed,
		Reason: unversioned.StatusReasonMethodNotAllowed,
		Details: &unversioned.StatusDetails{
			Kind: kind,
		},
		Message: fmt.Sprintf("%s is not supported on resources of kind %q", action, kind),
	}}
}

// NewServerTimeout returns an error indicating the requested action could not be completed due to a
// transient error, and the client should try again.
func NewServerTimeout(kind, operation string, retryAfterSeconds int) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusInternalServerError,
		Reason: unversioned.StatusReasonServerTimeout,
		Details: &unversioned.StatusDetails{
			Kind:              kind,
			Name:              operation,
			RetryAfterSeconds: retryAfterSeconds,
		},
		Message: fmt.Sprintf("The %s operation against %s could not be completed at this time, please try again.", operation, kind),
	}}
}

// NewInternalError returns an error indicating the item is invalid and cannot be processed.
func NewInternalError(err error) error {
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   http.StatusInternalServerError,
		Reason: unversioned.StatusReasonInternalError,
		Details: &unversioned.StatusDetails{
			Causes: []unversioned.StatusCause{{Message: err.Error()}},
		},
		Message: fmt.Sprintf("Internal error occurred: %v", err),
	}}
}

// NewTimeoutError returns an error indicating that a timeout occurred before the request
// could be completed.  Clients may retry, but the operation may still complete.
func NewTimeoutError(message string, retryAfterSeconds int) error {
	return &StatusError{unversioned.Status{
		Status:  unversioned.StatusFailure,
		Code:    StatusServerTimeout,
		Reason:  unversioned.StatusReasonTimeout,
		Message: fmt.Sprintf("Timeout: %s", message),
		Details: &unversioned.StatusDetails{
			RetryAfterSeconds: retryAfterSeconds,
		},
	}}
}

// NewGenericServerResponse returns a new error for server responses that are not in a recognizable form.
func NewGenericServerResponse(code int, verb, kind, name, serverMessage string, retryAfterSeconds int, isUnexpectedResponse bool) error {
	reason := unversioned.StatusReasonUnknown
	message := fmt.Sprintf("the server responded with the status code %d but did not return more information", code)
	switch code {
	case http.StatusConflict:
		if verb == "POST" {
			reason = unversioned.StatusReasonAlreadyExists
		} else {
			reason = unversioned.StatusReasonConflict
		}
		message = "the server reported a conflict"
	case http.StatusNotFound:
		reason = unversioned.StatusReasonNotFound
		message = "the server could not find the requested resource"
	case http.StatusBadRequest:
		reason = unversioned.StatusReasonBadRequest
		message = "the server rejected our request for an unknown reason"
	case http.StatusUnauthorized:
		reason = unversioned.StatusReasonUnauthorized
		message = "the server has asked for the client to provide credentials"
	case http.StatusForbidden:
		reason = unversioned.StatusReasonForbidden
		message = "the server does not allow access to the requested resource"
	case http.StatusMethodNotAllowed:
		reason = unversioned.StatusReasonMethodNotAllowed
		message = "the server does not allow this method on the requested resource"
	case StatusUnprocessableEntity:
		reason = unversioned.StatusReasonInvalid
		message = "the server rejected our request due to an error in our request"
	case StatusServerTimeout:
		reason = unversioned.StatusReasonServerTimeout
		message = "the server cannot complete the requested operation at this time, try again later"
	case StatusTooManyRequests:
		reason = unversioned.StatusReasonTimeout
		message = "the server has received too many requests and has asked us to try again later"
	default:
		if code >= 500 {
			reason = unversioned.StatusReasonInternalError
			message = "an error on the server has prevented the request from succeeding"
		}
	}
	switch {
	case len(kind) > 0 && len(name) > 0:
		message = fmt.Sprintf("%s (%s %s %s)", message, strings.ToLower(verb), kind, name)
	case len(kind) > 0:
		message = fmt.Sprintf("%s (%s %s)", message, strings.ToLower(verb), kind)
	}
	var causes []unversioned.StatusCause
	if isUnexpectedResponse {
		causes = []unversioned.StatusCause{
			{
				Type:    unversioned.CauseTypeUnexpectedServerResponse,
				Message: serverMessage,
			},
		}
	} else {
		causes = nil
	}
	return &StatusError{unversioned.Status{
		Status: unversioned.StatusFailure,
		Code:   code,
		Reason: reason,
		Details: &unversioned.StatusDetails{
			Kind: kind,
			Name: name,

			Causes:            causes,
			RetryAfterSeconds: retryAfterSeconds,
		},
		Message: message,
	}}
}

// IsNotFound returns true if the specified error was created by NewNotFound.
func IsNotFound(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonNotFound
}

// IsAlreadyExists determines if the err is an error which indicates that a specified resource already exists.
func IsAlreadyExists(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonAlreadyExists
}

// IsConflict determines if the err is an error which indicates the provided update conflicts.
func IsConflict(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonConflict
}

// IsInvalid determines if the err is an error which indicates the provided resource is not valid.
func IsInvalid(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonInvalid
}

// IsMethodNotSupported determines if the err is an error which indicates the provided action could not
// be performed because it is not supported by the server.
func IsMethodNotSupported(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonMethodNotAllowed
}

// IsBadRequest determines if err is an error which indicates that the request is invalid.
func IsBadRequest(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonBadRequest
}

// IsUnauthorized determines if err is an error which indicates that the request is unauthorized and
// requires authentication by the user.
func IsUnauthorized(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonUnauthorized
}

// IsForbidden determines if err is an error which indicates that the request is forbidden and cannot
// be completed as requested.
func IsForbidden(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonForbidden
}

// IsServerTimeout determines if err is an error which indicates that the request needs to be retried
// by the client.
func IsServerTimeout(err error) bool {
	return reasonForError(err) == unversioned.StatusReasonServerTimeout
}

// IsUnexpectedServerError returns true if the server response was not in the expected API format,
// and may be the result of another HTTP actor.
func IsUnexpectedServerError(err error) bool {
	switch t := err.(type) {
	case *StatusError:
		if d := t.Status().Details; d != nil {
			for _, cause := range d.Causes {
				if cause.Type == unversioned.CauseTypeUnexpectedServerResponse {
					return true
				}
			}
		}
	}
	return false
}

// IsUnexpectedObjectError determines if err is due to an unexpected object from the master.
func IsUnexpectedObjectError(err error) bool {
	_, ok := err.(*UnexpectedObjectError)
	return err != nil && ok
}

// SuggestsClientDelay returns true if this error suggests a client delay as well as the
// suggested seconds to wait, or false if the error does not imply a wait.
func SuggestsClientDelay(err error) (int, bool) {
	switch t := err.(type) {
	case *StatusError:
		if t.Status().Details != nil {
			switch t.Status().Reason {
			case unversioned.StatusReasonServerTimeout, unversioned.StatusReasonTimeout:
				return t.Status().Details.RetryAfterSeconds, true
			}
		}
	}
	return 0, false
}

func reasonForError(err error) unversioned.StatusReason {
	switch t := err.(type) {
	case *StatusError:
		return t.ErrStatus.Reason
	}
	return unversioned.StatusReasonUnknown
}
