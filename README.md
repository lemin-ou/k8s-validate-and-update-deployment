## K8s Hello Mutating Webhook
A Kubernetes Mutating Admission Webhook, using Go.
This is a companion repository for the Article [Automate ArgoCD deployment update](https://smartmssa.atlassian.net/l/cp/320KJHmm)


#### Run tests
```
$ make test
```

#### Deploy
Define shell env:
```
# define env vars
$ export CONTAINER_REPO=quay.io/my-user/my-repo
$ export CONTAINER_VERSION=x.y.z
```

Build/Push Webhook 
```
$ push-image
```
* for this you'll need to make the container repository public unless you'll be specifying ImagePullSecrets on the Pod

Deploy to K8s cluster
```
$ make k8s-deploy
```

#### Mutated pod example
```
$ k run busybox-1 --image=busybox  --restart=Never -l=hello=true -- sleep 3600
$ k exec busybox-1 -it -- ls /etc/config/hello.txt
# The output should be:
/etc/config/hello.txt
$ k exec busybox-1 -it -- sh -c "cat /etc/config/hello.txt"
# The output should be:
Hello from the admission controller !
```
We successfully mutated our pod spec and update the docker image tag using SSM parameter store, yay !

#### Cleanup
Delete all k8s resources
```
$ make k8s-delete-all
```

