/*
  Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
  Licensed under the Apache License, Version 2.0 (the "License").
  You may not use this file except in compliance with the License.
  A copy of the License is located at
      http://www.apache.org/licenses/LICENSE-2.0
  or in the "license" file accompanying this file. This file is distributed
  on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
  express or implied. See the License for the specific language governing
  permissions and limitations under the License.
*/

package handler

import (
	"k8s-update-deployment-ecr-tag/webhook/api/handler/function"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ssm"
	log "github.com/sirupsen/logrus"
)

func init() {
	loglvl := logLevels(os.Getenv("LOG_LEVEL")) // TODO: Set LOG_LEVEL environment variable
	log.SetFormatter(new(log.JSONFormatter))
	log.Infof("Got log level [%s]", loglvl)
	log.SetLevel(loglvl)
}

var (
	sess = session.Must(session.NewSession())

	svc    = ecr.New(sess, &aws.Config{Region: getRegistryRegion()})
	ssmSvc = ssm.New(sess, &aws.Config{Region: getRegistryRegion()})

	// Handler is the handler for the validating webhook.
	Handler = function.NewContainer(svc, ssmSvc).Handler().WithLogging()

	// Version is the shortened git hash of the binary's source code.
	// It is injected using the -X linker flag when running `make`
	Version string
)

// func main() {
// 	log.Infof("Starting function version: %s", Version)
// 	lambda.Start(Handler)
// }

func logLevels(lvl string) log.Level {
	loglvl, err := log.ParseLevel(lvl)
	if err != nil {
		return log.InfoLevel
	}

	return loglvl
}

func getRegistryRegion() *string {
	if value, ok := os.LookupEnv("REGISTRY_REGION"); ok {
		return aws.String(value)
	}
	return aws.String(os.Getenv("AWS_DEFAULT_REGION"))
}
