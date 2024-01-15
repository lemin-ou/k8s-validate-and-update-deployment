// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package function contains library units for the amazon-ecr-repository-compliance-webhook Lambda function.
package function

import (
	"context"
	"errors"
	"k8s-update-deployment-ecr-tag/webhook/api/handler/webhook"
	"net/http"

	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
)

// Errors returned when a validation expectation fails.
var (
	ErrFailedCompliance         = errors.New("webhook: repository fails ecr criteria")
	ErrImagesNotFound           = errors.New("webhook: no ecr images found in pod specification")
	ErrMultiImagesNotSuppported = errors.New("webhook: only one ecr image is needed for now !")
)

// Container contains the dependencies and business logic for the amazon-ecr-repository-compliance-webhook Lambda function.
type Container struct {
	ECR       ecriface.ECRAPI
	SSMClient SSMClient
}

// NewContainer creates a new function Container.
func NewContainer(ecrSvc ecriface.ECRAPI, ssmSvc ssmiface.SSMAPI) *Container {
	return &Container{
		ECR:       ecrSvc,
		SSMClient: *NewSSMClient(ssmSvc),
	}
}

// default HTTP status code to return on rejected admission
const code = 406          // NotAcceptable
const parameterCode = 407 // ParameterNotFound

// Handler returns the function handler for the amazon-ecr-repository-compliance-webhook.
// 1. Extract the POST request's body that ValidatingWebhookConfiguration admission controller made to API Gateway
// 2. Using the request, create a response. The response must contain the same UID that we received from the cluster
// 3. Using the request, extract the Pod object into the same Go data type used by Kubernetes
// 4. Using the Pod, check if the requested creation namespace is a critical one (e.g. default).
// 5. Using the Pod, extract all of the unique container images that are in the specification
//   - If no images in the specification come from ECR, deny the admission immediately
//
// 6. For every image provided, check our 4 requirements
// 7. If a single image didn't meet our requirements, deny the admission
// 8. All requirements satisfied, allow the Pod for admission
func (c *Container) Handler() Handler {
	return func(ctx context.Context, event *http.Request) (*v1beta1.AdmissionReview, error) {
		request, err := webhook.NewRequestFromEvent(event) // 1
		if err != nil {
			log.Errorf("Error creating request from event: %v", err)
			return webhook.BadRequestResponse(err)
		}

		response, err := webhook.NewResponseFromRequest(request) // 2
		if err != nil {
			log.Errorf("Error crafting response from request: %v", err)
			return webhook.BadRequestResponse(err)
		}

		pod, err := request.UnmarshalPod() // 3
		if err != nil {
			log.Errorf("Error unmarshalling Pod: %v", err)
			return response.FailValidation(code, err)
		}

		if webhook.InCriticalNamespace(pod) { // 4
			log.Info("Pod is in critical namespace, automatically passing")
			return response.PassValidation(""), nil
		}

		if webhook.NotInDeploymentNamespace(pod) { // 5
			log.Info("Pod is not in the deployment namespaces, automatically passing")
			return response.PassValidation(""), nil
		}

		registry, images := webhook.ParseImages(pod) // 6
		if len(images) == 0 {
			log.Error(ErrImagesNotFound)
			return response.FailValidation(code, ErrImagesNotFound)
		}
		if len(images) > 1 {
			log.Error(ErrMultiImagesNotSuppported)
			return response.FailValidation(code, ErrMultiImagesNotSuppported)
		}

		compliant, err := c.BatchCheckRepositoryCompliance(ctx, images) // 7
		if err != nil {
			log.Errorf("Error during compliance check: %v", err)
			return response.FailValidation(code, err)
		}

		if !compliant { // 8
			log.Error("Repository is not compliant")
			return response.FailValidation(code, ErrFailedCompliance)
		}

		// Now we only need to check for one repository, later it maybe more
		newImages, err := c.BatchUpdateImage(ctx, registry, images)
		if err != nil { // 9
			log.Errorf("Error during paramter fetching: %v", err)
			return response.FailValidation(parameterCode, err)
		}

		return response.PassValidation(newImages[0]), nil // 10
	}
}
