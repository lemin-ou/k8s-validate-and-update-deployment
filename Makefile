WEBHOOK_SERVICE?=k8s-update-deployment-ecr-tag
NAMESPACE?=kube-system
CONTAINER_REPO?=k8s-update-deployment-ecr-tag
CONTAINER_VERSION?=1.0.0
CONTAINER_REGISTRY?=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_DEFAULT_REGION).amazonaws.com
CONTAINER_IMAGE?=$(CONTAINER_REGISTRY)/$(CONTAINER_REPO):$(CONTAINER_VERSION)

# DOCKER STUFF

.PHONY: push-image
push-image: docker-build docker-login docker-push


.PHONY: docker-build
docker-build:
	docker build -t $(CONTAINER_REPO):$(CONTAINER_VERSION) webhook
	docker tag $(CONTAINER_REPO):$(CONTAINER_VERSION) $(CONTAINER_IMAGE)

.PHONY: docker-login
docker-login:
	aws ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(CONTAINER_REGISTRY)

.PHONY: docker-push
docker-push:
	docker push $(CONTAINER_IMAGE)




.PHONY: k8s-deploy
k8s-deploy: k8s-deploy-other k8s-deploy-csr k8s-deploy-deployment

.PHONY: k8s-deploy-other
k8s-deploy-other:
	kustomize build k8s/other | kubectl apply -f -
	kustomize build k8s/csr | kubectl apply -f -
	@echo Waiting for cert creation ...
	@sleep 15
	kubectl certificate approve $(WEBHOOK_SERVICE).$(NAMESPACE)

.PHONY: k8s-deploy-csr
k8s-deploy-csr:
	kustomize build k8s/csr | kubectl apply -f -
	@echo Waiting for cert creation ...
	@sleep 30
	kubectl certificate approve $(WEBHOOK_SERVICE).$(NAMESPACE)

.PHONY: k8s-deploy-deployment
k8s-deploy-deployment:
	(cd k8s/deployment && \
	kustomize edit set image CONTAINER_IMAGE=$(CONTAINER_IMAGE))
	kustomize build k8s/deployment | kubectl apply -f -

.PHONY: k8s-delete-all
k8s-delete-all:
	kustomize build k8s/other | kubectl delete --ignore-not-found=true -f  -
	kustomize build k8s/csr | kubectl delete --ignore-not-found=true -f  -
	kustomize build k8s/deployment | kubectl delete --ignore-not-found=true -f  -
	kubectl delete --ignore-not-found=true csr $(WEBHOOK_SERVICE).$(NAMESPACE)
	kubectl delete --ignore-not-found=true secret k8s-update-deployment-ecr-tag-secret

.PHONY: test
test:
	cd webhook && go test ./...