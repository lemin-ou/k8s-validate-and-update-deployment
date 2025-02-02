// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package function

import (
	"context"
	// "encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/admission/v1"
)

// Handler is a type alias for the Lambda handler's function signature.
type Handler func(context.Context, *http.Request) (*v1.AdmissionReview, error)

// ProxiedHandler is a handler that has been wrapped to respond with an API Gateway Proxy Integration.
// type ProxiedHandler func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// WithLogging is a logging middleware for the Lambda handler.
func (h Handler) WithLogging() Handler {
	return func(ctx context.Context, event *http.Request) (*v1.AdmissionReview, error) {
		review, err := h(ctx, event)
		log.Infof("Responding with AdmissionReview [%+v] and error [%v]", review, err)
		return review, err
	}
}

// WithProxiedResponse integrates the AdmissionReview response into an acceptable format
// for API Gateway proxy integrated Lambda functions.
// func (h Handler) WithProxiedResponse() ProxiedHandler {
// 	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// 		response := events.APIGatewayProxyResponse{
// 			Headers: map[string]string{"Content-Type": "application/json"},
// 		}
// 		review, err := h(ctx, event)
// 		if err != nil {
// 			response.Body = err.Error()
// 			response.StatusCode = 500
// 			return response, err
// 		}
// 		body, err := json.Marshal(review)
// 		if err != nil {
// 			response.Body = err.Error()
// 			response.StatusCode = 500
// 			return response, err
// 		}
// 		response.Body = string(body)
// 		response.StatusCode = 200 // Not to be confused with the status code in the AdmissionResponse
// 		return response, nil
// 	}
// }
