.PHONY: build
build: gobuilder
	docker build --build-arg "SERVICE_NAME=$(SERVICE_NAME)" \
	            -t "$(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG)" \
	            -t "$(DOCKER_IMAGE):latest" .

.PHONY: gobuilder
gobuilder:
	$(MAKE) build -C ../gobuilder
