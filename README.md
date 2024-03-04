## K8s Hello Mutating Webhook
A Kubernetes Mutating Admission Webhook, using Go.
This is a companion repository for the Article [Automate ArgoCD deployment update](https://smartmssa.atlassian.net/l/cp/320KJHmm)


#### Run tests
```
$ make test
```


Build/Push Webhook 
```
$ push-image
```
* for this you'll need to make the container repository public unless you'll be specifying ImagePullSecrets on the Pod spec

Deploy to K8s cluster
```
$ make k8s-deploy
```

#### Cleanup
Delete all k8s resources
```
$ make k8s-delete-all
```

