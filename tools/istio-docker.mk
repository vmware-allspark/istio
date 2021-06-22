## Copyright 2018 Istio Authors
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.

.PHONY: docker
.PHONY: docker.all
.PHONY: docker.save
.PHONY: docker.push

<<<<<<< HEAD
### Docker commands ###
# Below provides various commands to build/push docker images.
# These are all wrappers around ./tools/docker, the binary that controls docker builds.
# Builds can also be done through direct ./tools/docker invocations.
# When using these commands the flow is:
#  1) make target calls ./tools/docker
#  2) ./tools/docker calls `make build.docker.x` targets to compute the dependencies required
#  3) ./tools/docker triggers the actual docker commands required
# As a result, there are two layers of make involved.

docker: ## Build all docker images
	./tools/docker

docker.save: ## Build docker images and save to tar.gz
	./tools/docker --save

docker.push: ## Build all docker images and push to
	./tools/docker --push

# Legacy command aliases
docker.all: docker
	@:
dockerx.save: docker.save
	@:
dockerx.push: docker.push
	@:
dockerx.pushx: docker.push
	@:
dockerx: docker
	@:
=======
# Docker target will build the go binaries and package the docker for local testing.
# It does not upload to a registry.
docker: docker.all

# Add new docker targets to the end of the DOCKER_TARGETS list.

DOCKER_TARGETS ?= docker.pilot docker.proxyv2 docker.app docker.app_sidecar_ubuntu_xenial \
docker.app_sidecar_ubuntu_bionic docker.app_sidecar_ubuntu_focal docker.app_sidecar_debian_9 \
docker.app_sidecar_debian_10 docker.app_sidecar_centos_8 docker.app_sidecar_centos_7 \
docker.istioctl docker.operator docker.install-cni

# Echo docker directory and the template to pass image name and version to for VM testing
ECHO_DOCKER ?= pkg/test/echo/docker
VM_OS_DOCKERFILE_TEMPLATE ?= Dockerfile.app_sidecar

$(ISTIO_DOCKER) $(ISTIO_DOCKER_TAR):
	mkdir -p $@

.SECONDEXPANSION: #allow $@ to be used in dependency list

# directives to copy files to docker scratch directory

# tell make which files are copied from $(ISTIO_OUT_LINUX) and generate rules to copy them to the proper location:
# generates rules like the following:
# $(ISTIO_DOCKER)/pilot-agent: $(ISTIO_OUT_LINUX)/pilot-agent | $(ISTIO_DOCKER)
# 	cp $(ISTIO_OUT_LINUX)/$FILE $(ISTIO_DOCKER)/($FILE)
DOCKER_FILES_FROM_ISTIO_OUT_LINUX:=client server \
                             pilot-discovery pilot-agent \
                             istioctl manager
$(foreach FILE,$(DOCKER_FILES_FROM_ISTIO_OUT_LINUX), \
        $(eval $(ISTIO_DOCKER)/$(FILE): $(ISTIO_OUT_LINUX)/$(FILE) | $(ISTIO_DOCKER); cp $(ISTIO_OUT_LINUX)/$(FILE) $(ISTIO_DOCKER)/$(FILE)))

# rule for the test certs.
$(ISTIO_DOCKER)/certs:
	mkdir -p $(ISTIO_DOCKER)
	cp -a tests/testdata/certs $(ISTIO_DOCKER)/.
	chmod -R o+r $(ISTIO_DOCKER)/certs

# tell make which files are copied from the source tree and generate rules to copy them to the proper location:
# TODO(sdake)                      $(NODE_AGENT_TEST_FILES) $(GRAFANA_FILES)
DOCKER_FILES_FROM_SOURCE:=tests/testdata/certs/cert.crt tests/testdata/certs/cert.key
$(foreach FILE,$(DOCKER_FILES_FROM_SOURCE), \
        $(eval $(ISTIO_DOCKER)/$(notdir $(FILE)): $(FILE) | $(ISTIO_DOCKER); cp -p $(FILE) $$(@D)))

# BUILD_PRE tells $(DOCKER_RULE) to run the command specified before executing a docker build
# BUILD_ARGS tells  $(DOCKER_RULE) to execute a docker build with the specified commands

# The file must be named 'envoy', depends on the release.
${ISTIO_ENVOY_LINUX_RELEASE_DIR}/${SIDECAR}: ${ISTIO_ENVOY_LINUX_RELEASE_PATH}
	mkdir -p $(DOCKER_BUILD_TOP)/proxyv2
ifdef DEBUG_IMAGE
	cp ${ISTIO_ENVOY_LINUX_DEBUG_PATH} ${ISTIO_ENVOY_LINUX_RELEASE_DIR}/${SIDECAR}
else
	cp ${ISTIO_ENVOY_LINUX_RELEASE_PATH} ${ISTIO_ENVOY_LINUX_RELEASE_DIR}/${SIDECAR}
endif

# The file must be named 'envoy_bootstrap.json' because Dockerfile.proxyv2 hard-codes this.
${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/envoy_bootstrap.json: ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_PATH}
	cp ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_PATH} ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/envoy_bootstrap.json

# rule for wasm extensions.
$(ISTIO_ENVOY_LINUX_RELEASE_DIR)/stats-filter.wasm: init
$(ISTIO_ENVOY_LINUX_RELEASE_DIR)/stats-filter.compiled.wasm: init
$(ISTIO_ENVOY_LINUX_RELEASE_DIR)/metadata-exchange-filter.wasm: init
$(ISTIO_ENVOY_LINUX_RELEASE_DIR)/metadata-exchange-filter.compiled.wasm: init

# Default proxy image.
docker.proxyv2: BUILD_PRE=&& chmod 644 envoy_bootstrap.json gcp_envoy_bootstrap.json
docker.proxyv2: BUILD_ARGS=--build-arg proxy_version=istio-proxy:${PROXY_REPO_SHA} --build-arg istio_version=${VERSION} --build-arg BASE_VERSION=${BASE_VERSION} --build-arg SIDECAR=${SIDECAR}
docker.proxyv2: ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/envoy_bootstrap.json
docker.proxyv2: ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/gcp_envoy_bootstrap.json
docker.proxyv2: ${MODSEC_CONFIG_PATH}
docker.proxyv2: $(ISTIO_ENVOY_LINUX_RELEASE_DIR)/${SIDECAR}
docker.proxyv2: $(ISTIO_OUT_LINUX)/pilot-agent
docker.proxyv2: pilot/docker/Dockerfile.proxyv2
docker.proxyv2: $(ISTIO_ENVOY_LINUX_RELEASE_DIR)/stats-filter.wasm
docker.proxyv2: $(ISTIO_ENVOY_LINUX_RELEASE_DIR)/stats-filter.compiled.wasm
docker.proxyv2: $(ISTIO_ENVOY_LINUX_RELEASE_DIR)/metadata-exchange-filter.wasm
docker.proxyv2: $(ISTIO_ENVOY_LINUX_RELEASE_DIR)/metadata-exchange-filter.compiled.wasm
	$(DOCKER_RULE)

docker.pilot: BUILD_PRE=&& chmod 644 envoy_bootstrap.json gcp_envoy_bootstrap.json
docker.pilot: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.pilot: ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/envoy_bootstrap.json
docker.pilot: ${ISTIO_ENVOY_BOOTSTRAP_CONFIG_DIR}/gcp_envoy_bootstrap.json
docker.pilot: $(ISTIO_OUT_LINUX)/pilot-discovery
docker.pilot: pilot/docker/Dockerfile.pilot
	$(DOCKER_RULE)

# Test application
docker.app: BUILD_PRE=
docker.app: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.app: $(ECHO_DOCKER)/Dockerfile.app
docker.app: $(ISTIO_OUT_LINUX)/client
docker.app: $(ISTIO_OUT_LINUX)/server
docker.app: $(ISTIO_DOCKER)/certs
	$(DOCKER_RULE)

# Test application bundled with the sidecar with ubuntu:xenial (for non-k8s).
docker.app_sidecar_ubuntu_xenial: BUILD_ARGS=--build-arg VM_IMAGE_NAME=ubuntu --build-arg VM_IMAGE_VERSION=xenial --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_ubuntu_xenial: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_ubuntu_xenial: $(ISTIO_OUT_LINUX)/release/istio-sidecar.deb
docker.app_sidecar_ubuntu_xenial: $(ISTIO_DOCKER)/certs
docker.app_sidecar_ubuntu_xenial: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_ubuntu_xenial: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_ubuntu_xenial: $(ISTIO_OUT_LINUX)/server
	$(RENAME_TEMPLATE)
	$(DOCKER_RULE)

# Test application bundled with the sidecar with ubuntu:bionic (for non-k8s).
docker.app_sidecar_ubuntu_bionic: BUILD_ARGS=--build-arg VM_IMAGE_NAME=ubuntu --build-arg VM_IMAGE_VERSION=bionic --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_ubuntu_bionic: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_ubuntu_bionic: $(ISTIO_OUT_LINUX)/release/istio-sidecar.deb
docker.app_sidecar_ubuntu_bionic: $(ISTIO_DOCKER)/certs
docker.app_sidecar_ubuntu_bionic: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_ubuntu_bionic: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_ubuntu_bionic: $(ISTIO_OUT_LINUX)/server
	$(RENAME_TEMPLATE)
	$(DOCKER_RULE)

# Test application bundled with the sidecar with ubuntu:focal (for non-k8s).
docker.app_sidecar_ubuntu_focal: BUILD_ARGS=--build-arg VM_IMAGE_NAME=ubuntu --build-arg VM_IMAGE_VERSION=focal --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_ubuntu_focal: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_ubuntu_focal: $(ISTIO_OUT_LINUX)/release/istio-sidecar.deb
docker.app_sidecar_ubuntu_focal: $(ISTIO_DOCKER)/certs
docker.app_sidecar_ubuntu_focal: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_ubuntu_focal: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_ubuntu_focal: $(ISTIO_OUT_LINUX)/server
	$(RENAME_TEMPLATE)
	$(DOCKER_RULE)

# Test application bundled with the sidecar with debian 9 (for non-k8s).
docker.app_sidecar_debian_9: BUILD_ARGS=--build-arg VM_IMAGE_NAME=debian --build-arg VM_IMAGE_VERSION=9 --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_debian_9: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_debian_9: $(ISTIO_OUT_LINUX)/release/istio-sidecar.deb
docker.app_sidecar_debian_9: $(ISTIO_DOCKER)/certs
docker.app_sidecar_debian_9: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_debian_9: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_debian_9: $(ISTIO_OUT_LINUX)/server
	$(RENAME_TEMPLATE)
	$(DOCKER_RULE)

# Test application bundled with the sidecar with debian 10 (for non-k8s).
docker.app_sidecar_debian_10: BUILD_ARGS=--build-arg VM_IMAGE_NAME=debian --build-arg VM_IMAGE_VERSION=10 --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_debian_10: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_debian_10: $(ISTIO_OUT_LINUX)/release/istio-sidecar.deb
docker.app_sidecar_debian_10: $(ISTIO_DOCKER)/certs
docker.app_sidecar_debian_10: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_debian_10: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_debian_10: $(ISTIO_OUT_LINUX)/server
	$(RENAME_TEMPLATE)
	$(DOCKER_RULE)

# Test application bundled with the sidecar (for non-k8s).
docker.app_sidecar_centos_8: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION} --build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_centos_8: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_centos_8: $(ISTIO_OUT_LINUX)/release/istio-sidecar.rpm
docker.app_sidecar_centos_8: $(ISTIO_DOCKER)/certs
docker.app_sidecar_centos_8: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_centos_8: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_centos_8: $(ISTIO_OUT_LINUX)/server
docker.app_sidecar_centos_8: pkg/test/echo/docker/Dockerfile.app_sidecar_centos_8
	$(DOCKER_RULE)

# Test application bundled with the sidecar (for non-k8s).
docker.app_sidecar_centos_7: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.app_sidecar_centos_7: tools/packaging/common/envoy_bootstrap.json
docker.app_sidecar_centos_7: $(ISTIO_OUT_LINUX)/release/istio-sidecar-centos-7.rpm
docker.app_sidecar_centos_7: $(ISTIO_DOCKER)/certs
docker.app_sidecar_centos_7: pkg/test/echo/docker/echo-start.sh
docker.app_sidecar_centos_7: $(ISTIO_OUT_LINUX)/client
docker.app_sidecar_centos_7: $(ISTIO_OUT_LINUX)/server
docker.app_sidecar_centos_7: pkg/test/echo/docker/Dockerfile.app_sidecar_centos_7
	$(DOCKER_RULE)

docker.istioctl: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.istioctl: istioctl/docker/Dockerfile.istioctl
docker.istioctl: $(ISTIO_OUT_LINUX)/istioctl
	$(DOCKER_RULE)

docker.operator: manifests
docker.operator: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.operator: operator/docker/Dockerfile.operator
docker.operator: $(ISTIO_OUT_LINUX)/operator
	$(DOCKER_RULE)

# CNI
docker.install-cni: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION}
docker.install-cni: $(ISTIO_OUT_LINUX)/istio-cni
docker.install-cni: $(ISTIO_OUT_LINUX)/istio-iptables
docker.install-cni: $(ISTIO_OUT_LINUX)/install-cni
docker.install-cni: $(ISTIO_OUT_LINUX)/istio-cni-taint
docker.install-cni: cni/deployments/kubernetes/Dockerfile.install-cni
	$(DOCKER_RULE)

.PHONY: dockerx dockerx.save

# Can also be linux/arm64, or both with linux/amd64,linux/arm64
DOCKER_ARCHITECTURES ?= linux/amd64

# Docker has an experimental new build engine, https://github.com/docker/buildx
# This brings substantial (10x) performance improvements when building Istio
# However, its only built into docker since v19.03. Because its so new that devs are likely to not have
# this version, and because its experimental, this is not the default build method. As this matures we should migrate over.
# For performance, in CI this method is used.
# This target works by reusing the existing docker methods. Each docker target declares it's dependencies.
# We then override the docker rule and "build" all of these, where building just copies the dependencies
# We then generate a "bake" file, which defines all of the docker files in the repo
# Finally, we call `docker buildx bake` to generate the images.
dockerx: DOCKER_RULE?=mkdir -p $(DOCKERX_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./tools/docker-copy.sh $^ $(DOCKERX_BUILD_TOP)/$@ && cd $(DOCKERX_BUILD_TOP)/$@ $(BUILD_PRE)
dockerx: RENAME_TEMPLATE?=mkdir -p $(DOCKERX_BUILD_TOP)/$@ && cp $(ECHO_DOCKER)/$(VM_OS_DOCKERFILE_TEMPLATE) $(DOCKERX_BUILD_TOP)/$@/Dockerfile$(suffix $@)
dockerx: docker | $(ISTIO_DOCKER_TAR)
dockerx:
	HUBS="$(HUBS)" \
		TAG=$(TAG) \
		PROXY_REPO_SHA=$(PROXY_REPO_SHA) \
		VERSION=$(VERSION) \
		DOCKER_ALL_VARIANTS="$(DOCKER_ALL_VARIANTS)" \
		ISTIO_DOCKER_TAR=$(ISTIO_DOCKER_TAR) \
		INCLUDE_UNTAGGED_DEFAULT=$(INCLUDE_UNTAGGED_DEFAULT) \
		BASE_VERSION=$(BASE_VERSION) \
		DOCKERX_PUSH=$(DOCKERX_PUSH) \
		DOCKER_ARCHITECTURES=$(DOCKER_ARCHITECTURES) \
		./tools/buildx-gen.sh $(DOCKERX_BUILD_TOP) $(DOCKER_TARGETS)
	@# Retry works around https://github.com/docker/buildx/issues/298
	DOCKER_CLI_EXPERIMENTAL=enabled bin/retry.sh "read: connection reset by peer" docker buildx bake $(BUILDX_BAKE_EXTRA_OPTIONS) -f $(DOCKERX_BUILD_TOP)/docker-bake.hcl $(or $(DOCKER_BUILD_VARIANTS),default) || \
		{ tools/dump-docker-logs.sh; exit 1; }
>>>>>>> 38385ad6cb (TSM-2214 - Added modsec config to istio-proxy.)

# Support individual images like `dockerx.pilot`

# Docker commands defines some convenience targets
# Build individual docker image and push it. Ex: push.docker.pilot
push.docker.%:
	DOCKER_TARGETS=docker.$* ./tools/docker --push

# Build individual docker image and save it. Ex: tar.docker.pilot
tar.docker.%:
	DOCKER_TARGETS=docker.$* ./tools/docker --save

# Build individual docker image. Ex: docker.pilot
docker.%:
	DOCKER_TARGETS=docker.$* ./tools/docker

# Build individual docker image. Ex: dockerx.pilot
dockerx.docker.%:
	DOCKER_TARGETS=docker.$* ./tools/docker
### End docker commands ###
