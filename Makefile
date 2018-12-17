release:
	@echo "Latest 3 tags: "; \
	git ls-remote --tags | awk '{print $$2}' | sort -V | tail -n 3; \
	read -p "Enter New Tag:" tag; \
	echo "Releasing new tag: $$tag"; \
	git tag $$tag; \
	git push origin $$tag -f; \
	goreleaser --rm-dist

release-snapshot:
	goreleaser --snapshot --rm-dist

test:
	ginkgo -r

build-dev-dockerfile:
	docker build . -f docker/dev/Dockerfile

build:
	goreleaser --skip-publish --skip-sign --skip-validate --rm-dist