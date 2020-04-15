.PHONY: build
build:
	cwd="$$(pwd)"; cd $$(git rev-parse --show-toplevel) \
	&& docker build --file "$$cwd/Dockerfile" -t "localhost/gobuilder:latest" .
