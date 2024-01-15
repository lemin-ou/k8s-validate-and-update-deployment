package function

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	log "github.com/sirupsen/logrus"
)

const seperator = "-"

// Due to our naming convension of repository "project-ptype", e.g.
// From project-ptype to /project/ptype/tag. ptype can be : frontend, backend.
func reconstruct(repo string) string {
	// if !strings.Contains(repo, seperator) {
	// 	return nil,
	// }
	segments := strings.Split(repo, seperator)
	project, ptype := segments[0], segments[1]
	log.Tracef("parts: project [%s], type [%s]", project, ptype)
	return fmt.Sprintf("/%s/%s/ecr_tag", project, ptype)
}

// SSMClient created to use the SSM API
type SSMClient struct {
	SSM ssmiface.SSMAPI
}

// NewSSMClient creates a new SSMClient
func NewSSMClient(ssmSvc ssmiface.SSMAPI) *SSMClient {
	return &SSMClient{
		SSM: ssmSvc,
	}
}
