LDFLAGS=-w -s

.PHONY: build
build:
	go build -v -ldflags="$(LDFLAGS)" -o "$(SERVICE_NAME)" -mod vendor

.PHONY: export
export:
	@if [ ! -z "$(SERVICE_EXPORT_DIRS)" ]; then mkdir -p "./exported" && cp -r $(SERVICE_EXPORT_DIRS) "./exported"; fi

.PHONY: docker/build
docker/build:
	$(MAKE) build -C ../../docker/service-torsocks \
		SERVICE_NAME=$(SERVICE_NAME) \
		DOCKER_IMAGE=$(DOCKER_IMAGE) \
		DOCKER_IMAGE_TAG=$(DOCKER_IMAGE_TAG)
