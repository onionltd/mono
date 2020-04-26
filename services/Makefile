LDFLAGS=-w -s

GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT_HASH=$(shell git rev-parse --short HEAD)
DOCKER_IMAGE_TAG=$(GIT_BRANCH)-$(GIT_COMMIT_HASH)

.PHONY: build
build:
	go build -v -ldflags="$(LDFLAGS)" -o "$(SERVICE_NAME)" -mod vendor

.PHONY: export
export:
	@mkdir -p "./exported" && if [ ! -z "$(SERVICE_EXPORT_DIRS)" ]; then cp -r $(SERVICE_EXPORT_DIRS) "./exported"; fi

.PHONY: docker/build
docker/build:
	$(MAKE) build -C ../../docker/service-torsocks \
		SERVICE_NAME=$(SERVICE_NAME) \
		DOCKER_IMAGE=$(DOCKER_IMAGE) \
		DOCKER_IMAGE_TAG=$(DOCKER_IMAGE_TAG)

.PHONY: docker/save
docker/save:
	docker save -o "$(SERVICE_NAME):$(DOCKER_IMAGE_TAG).tar" $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG)
