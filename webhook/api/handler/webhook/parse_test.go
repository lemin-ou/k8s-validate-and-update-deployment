// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"reflect"
	"testing"

	"k8s-update-deployment-ecr-tag/webhook/api/testdata"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestParseRepositories(t *testing.T) {
	var (
		untaggedImageDeployment   = newDeploymentWithImage(testdata.UntaggedImage)
		taggedImageDeployment     = newDeploymentWithImage(testdata.TaggedImage)
		cnImageDeployment         = newDeploymentWithImage(testdata.CNImage)
		fipsImageDeployment       = newDeploymentWithImage(testdata.FIPSImage)
		duplicateImagesDeployment = newDeploymentWithImage(testdata.TaggedImage)
		twoImagesDeployment       = newDeploymentWithImage(testdata.TaggedImage)
		noNamespaceDeployment     = newDeploymentWithImage(testdata.NoNamespace)
		aliasedImageDeployment    = newDeploymentWithImage(testdata.AliasedImage)
		noImages                  = newDeploymentWithImage("")
		badImage                  = newDeploymentWithImage("elgoog/sselortsid")
	)
	duplicateImagesDeployment.Spec.Template.Spec.Containers = append(duplicateImagesDeployment.Spec.Template.Spec.Containers, duplicateImagesDeployment.Spec.Template.Spec.Containers...)
	twoImagesDeployment.Spec.Template.Spec.Containers = append(twoImagesDeployment.Spec.Template.Spec.Containers, untaggedImageDeployment.Spec.Template.Spec.Containers...)
	tests := []struct {
		name       string
		deployment *appsv1.Deployment
		want       []string
	}{
		{"UntaggedImage", untaggedImageDeployment, []string{"namespace/repo@sha256:e5e2a3236e64483c50dd2811e46e9cd49c67e82271e60d112ca69a075fc23005"}},
		{"TaggedImage", taggedImageDeployment, []string{"namespace/repo:40d6072"}},
		{"CNImage", cnImageDeployment, []string{"namespace/repo:40d6072"}},
		{"FIPSImage", fipsImageDeployment, []string{"namespace/repo:40d6072"}},
		{"Duplicates", duplicateImagesDeployment, []string{"namespace/repo:40d6072"}},
		{"TwoImages", twoImagesDeployment, []string{"namespace/repo:40d6072", "namespace/repo@sha256:e5e2a3236e64483c50dd2811e46e9cd49c67e82271e60d112ca69a075fc23005"}},
		{"NoNamespace", noNamespaceDeployment, []string{"repo:40d6072"}},
		{"Aliased", aliasedImageDeployment, []string{"namespace/repo:40d6072"}},
		{"NoImages", noImages, nil},
		{"BadImage", badImage, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, got := ParseImages(tt.deployment); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRepositories() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newDeploymentWithImage(image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: image,
						},
					},
				},
			},
		},
	}
}
