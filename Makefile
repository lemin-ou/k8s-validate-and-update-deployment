WEBHOOK_SERVICE?=k8s-update-deployment-ecr-tag
NAMESPACE?=kube-system
CONTAINER_REPO?=k8s-update-deployment-ecr-tag
CONTAINER_VERSION?=1.0.6
CONTAINER_REGISTRY?=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_DEFAULT_REGION).amazonaws.com
CONTAINER_IMAGE?=$(CONTAINER_REGISTRY)/$(CONTAINER_REPO):$(CONTAINER_VERSION)
TEST_CONTAINER_IMAGE_1=test-frontend
TEST_CONTAINER_IMAGE_1_TAG=latest
TEST_CONTAINER_IMAGE_2=test2-frontend
TEST_CONTAINER_IMAGE_2_TAG=bec0e8f
TEST_CONTAINER_IMAGE_1_SSM_P=/test2/frontend/ecr_tag
WEBHOOK=$(WEBHOOK_SERVICE).smartdev.ai
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
	aws ecr create-repository --region ${AWS_DEFAULT_REGION} --repository-name $(CONTAINER_REPO) 2>&1 | grep -q "RepositoryAlreadyExistsException" && echo "Repository already exists." || echo "Repository created successfully."


.PHONY: docker-push
docker-push:
	docker push $(CONTAINER_IMAGE)




.PHONY: k8s-deploy
k8s-deploy: k8s-deploy-other k8s-deploy-deployment k8s-patch-webhook

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

.PHONY: k8s-patch-webhook
k8s-patch-webhook:
	@echo "Trying to patch webhook removing objectSelector"; 
	kubectl patch mutatingwebhookconfiguration "$(WEBHOOK)" --type='json' -p "[{'op': 'remove', 'path': '/webhooks/0/objectSelector'}]"

.PHONY: k8s-patch-webhook-add-objectSelector
k8s-patch-webhook-add-objectSelector:
	@echo "Trying to patch webhook adding objectSelector"; 
	kubectl patch mutatingwebhookconfiguration "$(WEBHOOK)" --type='json' -p '[{"op": "add", "path": "/webhooks/0/objectSelector/matchLabels", "value": {"stop-from-executing": "true"}}]'


.PHONY: k8s-delete-all
k8s-delete-all:
	kustomize build k8s/other | kubectl delete --ignore-not-found=true -f  -
	kustomize build k8s/csr | kubectl delete --ignore-not-found=true -f  -
	kustomize build k8s/deployment | kubectl delete --ignore-not-found=true -f  -
	kubectl delete --ignore-not-found=true csr $(WEBHOOK_SERVICE).$(NAMESPACE)
	kubectl delete --ignore-not-found=true secret k8s-update-deployment-ecr-tag-secret

.PHONY: prepare-test
prepare-test:
	cd webhook/api/testdata
	docker build -t $(TEST_CONTAINER_IMAGE_1):$(TEST_CONTAINER_IMAGE_1_TAG) webhook/api/testdata
	docker build -t $(TEST_CONTAINER_IMAGE_2):$(TEST_CONTAINER_IMAGE_2_TAG) webhook/api/testdata
	aws ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(CONTAINER_REGISTRY)
	aws ecr create-repository --region ${AWS_DEFAULT_REGION} --repository-name $(TEST_CONTAINER_IMAGE_1) 2>&1 | grep -q "RepositoryAlreadyExistsException" && echo "Repository already exists." || echo "Repository created successfully."
	aws ecr create-repository --region ${AWS_DEFAULT_REGION} --repository-name $(TEST_CONTAINER_IMAGE_2) 2>&1 | grep -q "RepositoryAlreadyExistsException" && echo "Repository already exists." || echo "Repository created successfully."
	docker tag $(TEST_CONTAINER_IMAGE_1):$(TEST_CONTAINER_IMAGE_1_TAG) $(CONTAINER_REGISTRY)/$(TEST_CONTAINER_IMAGE_1):$(TEST_CONTAINER_IMAGE_1_TAG)
	docker tag $(TEST_CONTAINER_IMAGE_2):$(TEST_CONTAINER_IMAGE_2_TAG) $(CONTAINER_REGISTRY)/$(TEST_CONTAINER_IMAGE_2):$(TEST_CONTAINER_IMAGE_2_TAG)
	docker push $(CONTAINER_REGISTRY)/$(TEST_CONTAINER_IMAGE_1):$(TEST_CONTAINER_IMAGE_1_TAG)
	docker push $(CONTAINER_REGISTRY)/$(TEST_CONTAINER_IMAGE_2):$(TEST_CONTAINER_IMAGE_2_TAG)

	aws ssm put-parameter --type String --name $(TEST_CONTAINER_IMAGE_1_SSM_P) --description "this parameter is for testing the k8s-update-deployment-ecr-tag project, DO-NOT-DELETE" --value $(TEST_CONTAINER_IMAGE_2_TAG) --region $(AWS_DEFAULT_REGION) 

.PHONY: test
test:
   # The following images must exists in ECR in Paris (eu-west-3) region: test-backend, test-frontend
   # The followng SSM Parameter Store must also exist in Paris (eu-west-3) region: /test/frontend/ecr_tag with the value of bec0e8f
   # to prepare the environment for the tests call the prepare-test PHONY before this PHONY
   # If not the tests won't pass.
	cd webhook && go test ./...