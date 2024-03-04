// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s-update-deployment-ecr-tag/webhook/api"
	"k8s-update-deployment-ecr-tag/webhook/api/testdata"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockECRClient struct {
	mock.Mock
	ecriface.ECRAPI
}

// DescribeRepositoriesWithContext mocks the DescribeRepositories ECR API endpoint.
func (_m *mockECRClient) DescribeRepositoriesWithContext(ctx aws.Context, input *ecr.DescribeRepositoriesInput, opts ...request.Option) (*ecr.DescribeRepositoriesOutput, error) {
	log.Infof("Mocking DescribeRepositories API with input: %s\n", input.String())
	args := _m.Called(ctx, input)
	return args.Get(0).(*ecr.DescribeRepositoriesOutput), args.Error(1)
}

// DescribeImageScanFindingsPagesWithContext mocks the DescribeImageScanFindingsP ECR API endpoint.
func (_m *mockECRClient) DescribeImageScanFindingsPagesWithContext(ctx aws.Context, input *ecr.DescribeImageScanFindingsInput, fn func(*ecr.DescribeImageScanFindingsOutput, bool) bool, opts ...request.Option) error {
	log.Debugf("Mocking DescribeImageScanFindings API with input: %s\n", input.String())
	args := _m.Called(ctx, input, fn)
	return args.Error(0)
}

var s *httptest.Server
var patchType = v1.PatchTypeJSONPatch

func TestHandler(t *testing.T) {

	type args struct {
		image           string
		repo            *ecr.Repository
		shouldCheckVuln bool
		scanFindings    *ecr.DescribeImageScanFindingsOutput
		event           http.Request
	}

	type patch struct {
		patchType *v1.PatchType
		value     []byte
	}

	ecrSvc := new(mockECRClient)

	app := &api.App{}

	r := api.BuildRouter(app)
	s = httptest.NewServer(r)

	url, err := url.Parse(s.URL + "/")
	require.Nil(t, err)

	req := http.Request{URL: url, Method: "POST", Header: map[string][]string{"Content-Type": {"application/json"}}}

	defer s.Close()

	tests := []struct {
		name    string
		args    args
		status  string
		wantErr bool
		patch   patch
	}{
		{
			name: "BadRequestFailure",
			args: args{
				shouldCheckVuln: false,
				repo:            nil,
				event:           eventWithBadRequest(req),
			},
			status:  metav1.StatusFailure,
			wantErr: true,
		},
		{
			name: "BadRequestNoUIDFailure",
			args: args{
				shouldCheckVuln: false,
				repo:            nil,
				event:           eventWithNoUID(req),
			},
			status:  metav1.StatusFailure,
			wantErr: true,
		},
		{
			name: "NonExistingRepository",
			args: args{
				image:           "auth:notlatest",
				shouldCheckVuln: false,
				repo:            nil,
				event:           eventWithImage(req, "123456789012.dkr.ecr.region.amazonaws.com/auth:notlatest"),
			},
			status:  metav1.StatusFailure,
			wantErr: true,
		},
		{
			name: "NotECRRepositoryFailure",
			args: args{
				image:           "nginx-ingress-controller:0.30.0",
				shouldCheckVuln: false,
				repo:            nil,
				event:           eventWithImage(req, "quay.io/kubernetes-ingress-controller/nginx-ingress-controller:0.30.0"),
			},
			status:  metav1.StatusFailure,
			wantErr: true,
		},
		{
			name: "ExistingRepositoryWithNoParameterStoreParameter",
			args: args{
				image:           "test-frontend",
				shouldCheckVuln: false,
				repo: &ecr.Repository{
					RepositoryName:             aws.String("test-frontend"),
					ImageTagMutability:         aws.String(ecr.ImageTagMutabilityMutable),
					ImageScanningConfiguration: &ecr.ImageScanningConfiguration{ScanOnPush: aws.Bool(false)},
				},
				event: eventWithImage(req, "123456789012.dkr.ecr.region.amazonaws.com/test-frontend:notlatest"),
			},
			status:  metav1.StatusFailure,
			wantErr: true,
		},
		{
			name: "ExistAndWithSSMParameterStoreParameter",
			args: args{
				image:           "test2-frontend",
				shouldCheckVuln: false,
				repo: &ecr.Repository{
					RepositoryName:             aws.String("test2-frontend"),
					ImageTagMutability:         aws.String(ecr.ImageTagMutabilityMutable),
					ImageScanningConfiguration: &ecr.ImageScanningConfiguration{ScanOnPush: aws.Bool(false)},
				},
				event: eventWithImage(req, "123456789012.dkr.ecr.region.amazonaws.com/test2-frontend:notlatest"),
			},
			patch: patch{
				patchType: &patchType,
				// base64 encoded : '[{"op":"replace","path":"/spec/containers/0/image", "value": "123456789012.dkr.ecr.region.amazonaws.com/test2-frontend:bec0e8f"}]'
				value: []byte("[{\"op\":\"replace\",\"path\":\"/spec/template/spec/containers/0/image\", \"value\": \"123456789012.dkr.ecr.region.amazonaws.com/test2-frontend:bec0e8f\"}]"),
			},
			status:  metav1.StatusSuccess,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ctx, cancel := context.WithCancel(context.Background())
			// defer cancel()
			// if tt.args.repo != nil {
			// 	ecrSvc.On("DescribeRepositoriesWithContext",
			// 		ctx,
			// 		&ecr.DescribeRepositoriesInput{
			// 			RepositoryNames: []*string{tt.args.repo.RepositoryName},
			// 		},
			// 	).Return(&ecr.DescribeRepositoriesOutput{Repositories: []*ecr.Repository{tt.args.repo}}, nil)
			// }

			// Deactivate those tests for now, until their code is activated
			// if tt.args.shouldCheckVuln {
			// 	// DescribeImageScanFindingsPagesWithContext will not be called if
			// 	// the repository fails one of the first three checks
			// 	tag := strings.Split(tt.args.image, ":")[1]
			// 	ecrSvc.On("DescribeImageScanFindingsPagesWithContext",
			// 		ctx,
			// 		&ecr.DescribeImageScanFindingsInput{
			// 			ImageId: &ecr.ImageIdentifier{
			// 				ImageTag: aws.String(tag),
			// 			},
			// 			RepositoryName: tt.args.repo.RepositoryName,
			// 		},
			// 		mock.AnythingOfType("func(*ecr.DescribeImageScanFindingsOutput, bool) bool"),
			// 	).Return(nil).Run(func(args mock.Arguments) {
			// 		arg := args.Get(2).(func(*ecr.DescribeImageScanFindingsOutput, bool) bool)
			// 		arg(tt.args.scanFindings, true)
			// 	})
			// }
			// review, err := h(context.Background(), &tt.args.event)

			resp, err := http.DefaultClient.Do(&tt.args.event)

			require.NoError(t, err)

			if err != nil {
				t.Fatalf("Error during request for image: %v", err)
			}

			var review v1.AdmissionReview

			buffer, err := io.ReadAll(resp.Body)
			json.Unmarshal(buffer, &review)
			t.Logf("Got review body: %#+v", review)
			require.Nil(t, err)
			require.Equal(t, tt.status, review.Response.Result.Status)
			if tt.wantErr {
				require.GreaterOrEqual(t, review.Response.Result.Code, int32(400))
				require.Less(t, review.Response.Result.Code, int32(500))
			}
			if tt.status == metav1.StatusSuccess {
				require.GreaterOrEqual(t, review.Response.Result.Code, int32(200))
				require.Equal(t, review.Response.PatchType, tt.patch.patchType)
				require.Equal(t, review.Response.Patch, tt.patch.value)
			}
			ecrSvc.AssertExpectations(t)
		})
	}
}

func eventWithNoUID(req http.Request) http.Request {
	req.Body = io.NopCloser(strings.NewReader(testdata.ReviewWithNoUID))
	return req
}

func eventWithBadRequest(req http.Request) http.Request {
	req.Body = io.NopCloser(strings.NewReader(testdata.ReviewWithBadRequest))
	return req
}

func eventWithImage(req http.Request, image string) http.Request {
	deploymentNamespace := os.Getenv("DEPLOYMENT_NAMESPACE")
	req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(testdata.ReviewWithOneImage, deploymentNamespace, deploymentNamespace, image)))
	return req
}

func findingsWithCriticalVuln() *ecr.DescribeImageScanFindingsOutput {
	return &ecr.DescribeImageScanFindingsOutput{
		ImageScanFindings: &ecr.ImageScanFindings{
			Findings: []*ecr.ImageScanFinding{
				{
					Severity: aws.String(ecr.FindingSeverityCritical),
				},
			},
		},
	}
}

func findingsWithNoVuln() *ecr.DescribeImageScanFindingsOutput {
	return &ecr.DescribeImageScanFindingsOutput{
		ImageScanFindings: &ecr.ImageScanFindings{
			Findings: []*ecr.ImageScanFinding{
				{
					Severity: aws.String(ecr.FindingSeverityInformational),
				},
			},
		},
	}
}
