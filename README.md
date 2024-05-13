## K8s Hello Mutating Webhook
A Kubernetes Mutating Admission Webhook, using Go, that intercept deployement and checks the validy of ECR images (in deployment's pod specification) and update deployment image tag using a specific SSM Parameter Store.

This K8s Mutating Admission Webhook perform the following:

- Retreive the ECR image from the deployment
- Check the existance and validy of the ECR repository and image
- Retreive the value of a SSM paramter store (that have a specific path -> `/{PROJECT_I}/{frontend or backend}/ecr_tag`. eg: */gmt/frontend/ecr_tag*, */gmt/backend/ecr_tag*). The value of the parameter is an ECR repository tag (this would be a the tag stored from a previous CI/CD pipeline execution).
- Update the deployemnt image with the value of the SSM Parameter Store.

#### Run tests
```
$ make test
```


Build/Push Webhook 
```
$ push-image
```

Deploy to K8s cluster
```
$ make k8s-deploy
```

#### Cleanup
Delete all k8s resources
```
$ make k8s-delete-all
```

