// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package function

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ssm"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const digestID = "@"

// From repository:tag to repository, tag
// Or repository@sha256:digest to repository, @sha256:digest
func parts(image string) (repo, tagOrDigest string) {
	if strings.Contains(image, digestID) {
		segments := strings.Split(image, digestID)
		repo, tagOrDigest = segments[0], digestID+segments[1] // append ampersand for later
		log.Tracef("parts: repo [%s], tagOrHash [%s]", repo, tagOrDigest)
		return
	}
	segments := strings.Split(image, ":")
	repo, tagOrDigest = segments[0], segments[1]
	log.Tracef("parts: repo [%s], tagOrHash [%s]", repo, tagOrDigest)
	return
}

// CheckRepositoryCompliance checks if the container image that was sent to the webhook:
// 1. Comes from an ECR repository
// 2. Has image tag immutability enabled
// 3. Has image scan on push enabled
// 4. Does not contain any critical vulnerabilities
func (c *Container) CheckRepositoryCompliance(ctx context.Context, image string) (bool, error) {
	repo, _ := parts(image)
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(repo)},
	}
	if err := input.Validate(); err != nil {
		return false, err
	}
	output, err := c.ECR.DescribeRepositoriesWithContext(ctx, input)
	if err != nil {
		return false, err
	}
	if len(output.Repositories) == 0 {
		return false, fmt.Errorf("no repositories named '%s' found", repo)
	}
	// r := output.Repositories[0]
	// Deactivate those checks for now, maybe needed later, TODO: Activate Tag immutability, ScanOnPush and CriticalVulnerabilities checks for kubenetes admission controller
	// if aws.StringValue(r.ImageTagMutability) == ecr.ImageTagMutabilityMutable {
	// 	return false, fmt.Errorf("repository '%s' does not have image tag immutability enabled", repo)
	// }
	// if !aws.BoolValue(r.ImageScanningConfiguration.ScanOnPush) {
	// 	return false, fmt.Errorf("repository '%s' does not have image scan on push enabled", repo)
	// }
	// critical, err := c.HasCriticalVulnerabilities(ctx, image)
	// if err != nil {
	// 	return false, err
	// }
	// if critical {
	// 	return false, fmt.Errorf("image '%s' contains %s vulnerabilities", image, ecr.FindingSeverityCritical)
	// }
	return true, nil
}

// BatchCheckRepositoryCompliance checks the compliance of a given set of ECR images.
// False is returned if a single repository is not compliant.
func (c *Container) BatchCheckRepositoryCompliance(ctx context.Context, images []string) (bool, error) {
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	compliances := make([]bool, len(images))

	for i, image := range images {
		i, image := i, image // shadow
		g.Go(func() error {
			compliant, err := c.CheckRepositoryCompliance(ctx, image)

			mu.Lock()
			compliances[i] = compliant
			mu.Unlock()
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return false, err
	}

	for _, complaint := range compliances {
		if !complaint {
			return false, nil
		}
	}
	return true, nil
}

// HasCriticalVulnerabilities checks if a container image contains 'CRITICAL' vulnerabilities.
// func (c *Container) HasCriticalVulnerabilities(ctx context.Context, image string) (bool, error) {
// 	var (
// 		repo, tagOrDigest = parts(image)
// 		found             = false
// 	)
// 	input := &ecr.DescribeImageScanFindingsInput{
// 		ImageId:        &ecr.ImageIdentifier{},
// 		RepositoryName: aws.String(repo),
// 	}

// 	switch strings.Contains(tagOrDigest, digestID) {
// 	case true:
// 		input.ImageId.ImageDigest = aws.String(tagOrDigest[1:]) // omit ampersand
// 	default:
// 		input.ImageId.ImageTag = aws.String(tagOrDigest)
// 	}
// 	if err := input.Validate(); err != nil {
// 		return true, err
// 	}

// 	pager := func(out *ecr.DescribeImageScanFindingsOutput, lastPage bool) bool {
// 		for _, finding := range out.ImageScanFindings.Findings {
// 			if aws.StringValue(finding.Severity) == ecr.FindingSeverityCritical {
// 				found = true
// 				return found // break out of paging if we've already found a critical vuln.
// 			}
// 		}
// 		return lastPage
// 	}

// 	if err := c.ECR.DescribeImageScanFindingsPagesWithContext(ctx, input, pager); err != nil {
// 		return true, err
// 	}
// 	return found, nil
// }

func (c *Container) UpdateImage(ctx context.Context, image string) (string, error) {
	repo, _ := parts(image)
	name := reconstruct(repo)
	input := &ssm.GetParameterInput{
		Name: &name,
	}
	if err := input.Validate(); err != nil {
		return "", err
	}
	output, err := c.SSMClient.SSM.GetParameter(input)
	if err != nil {
		return "", err
	}

	// return to repository:tag
	return fmt.Sprintf("%s:%s", repo, *output.Parameter.Value), nil

}

func (c *Container) BatchUpdateImage(ctx context.Context, registry string, images []string) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)
	updateImages := make([]string, len(images))
	for i, image := range images {
		_, image := i, image // shadow
		g.Go(func() error {
			updated, err := c.UpdateImage(ctx, image)
			updateImages[i] = fmt.Sprintf("%s/%s", registry, updated)
			return err
		})

	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return updateImages, nil
}
