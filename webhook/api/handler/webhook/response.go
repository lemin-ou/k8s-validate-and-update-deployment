// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Errors returned when a bad request is received or a failure reason is not provided.
var (
	ErrMissingFailure = errors.New("webhook: reached invalid state, no failure reason found")
	ErrBadRequest     = errors.New("webhook: bad request")
)

const patchUpdateImage = `[{"op":"replace","path":"/spec/template/spec/containers/0/image", "value": "%s"}]`

// BadRequestResponse is the response returned to the cluster when a bad request is sent.
func BadRequestResponse(err error) (*v1.AdmissionReview, error) {
	response := &v1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: err.Error(),
			Reason:  metav1.StatusReasonBadRequest,
			Code:    400,
		},
	}
	return respond(response), nil
}

// Response encapsulates the AdmissionResponse sent to API Gateway.
type Response struct {
	Admission *v1.AdmissionResponse
}

// NewResponseFromRequest creates a Response from a Request.
func NewResponseFromRequest(r *Request) (*Response, error) {
	if r == nil || r.Admission == nil {
		return nil, ErrBadRequest
	}
	if r.Admission != nil && r.Admission.UID == "" {
		return nil, ErrBadRequest
	}
	return &Response{
		Admission: &v1.AdmissionResponse{
			UID: r.Admission.UID,
		},
	}, nil
}

// FailValidation populates the AdmissionResponse with the failure contents
// (message and error) and returns the AdmissionReview JSON body response for API Gateway.
func (r *Response) FailValidation(code int32, failure error) (*v1.AdmissionReview, error) {
	if failure == nil {
		return nil, ErrMissingFailure
	}

	r.Admission.Allowed = false
	r.Admission.Result = &metav1.Status{
		Status:  metav1.StatusFailure,
		Message: failure.Error(),
		// Need a better way to Code with Reason; maybe use gRPC code mappings?
		Reason: metav1.StatusReasonNotAcceptable,
		Code:   code,
	}
	return respond(r.Admission), nil
}

// PassValidation populates the AdmissionResponse with the pass contents
// (message) and returns the AdmissionReview JSON response for API Gateway.
func (r *Response) PassValidation(image string) *v1.AdmissionReview {
	r.Admission.Allowed = true
	// Mutating the AdmissionReview
	if len(image) != 0 {
		patchType := v1.PatchTypeJSONPatch
		r.Admission.PatchType = &patchType
		r.Admission.Patch = encodePatch(image)
	}
	r.Admission.Result = &metav1.Status{
		Status:  metav1.StatusSuccess,
		Message: "deployment contains compliant ecr repositories and images",
		Code:    200,
	}
	return respond(r.Admission)
}

func respond(admission *v1.AdmissionResponse) *v1.AdmissionReview {
	return &v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admission,
	}
}

func encodePatch(image string) []byte {
	fmt.Printf("final patch: %s", fmt.Sprintf(patchUpdateImage, image))
	patchBytes := []byte(fmt.Sprintf(patchUpdateImage, image))

	// Create a byte slice to store the base64-encoded result
	// encodedBytes := make([]byte, base64.StdEncoding.EncodedLen(len(patchBytes)))
	// base64.StdEncoding.Encode(encodedBytes, patchBytes)
	return patchBytes
}
