ARG GO_BUILDER=brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.24
ARG RUNTIME=scratch
# Add FIPS compliance layer
ARG FIPS_BUILDER=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:bb08f2300cb8d12a7eb91dddf28ea63692b3ec99e7f0fa71a1b300f2756ea829
FROM $FIPS_BUILDER AS fips_builder

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/pipeline
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD
ENV GOEXPERIMENT=strictfipsruntime
ENV GODEBUG="http2server=0"
RUN go build -ldflags="-X 'knative.dev/pkg/changeset.rev=$(cat HEAD)'" -mod=vendor -tags disable_gcp,strictfipsruntime -v -o /tmp/nop \
    ./cmd/nop

FROM $RUNTIME
ARG VERSION=pipeline-next

ENV NOP=/usr/local/bin/nop \
    KO_APP=/ko-app \
    KO_DATA_PATH=/kodata

COPY --from=builder /tmp/nop /ko-app/nop
COPY head ${KO_DATA_PATH}/HEAD

# Copy FIPS-compliant libraries
COPY --from=fips_builder /usr/lib64/libcrypto.so.3 /usr/lib64/
COPY --from=fips_builder /usr/lib64/libcrypto.so.3.5.1 /usr/lib64/
COPY --from=fips_builder /usr/lib64/ossl-modules/fips.so /usr/lib64/ossl-modules/

LABEL \
      com.redhat.component="openshift-pipelines-nop-rhel9-container" \
      name="openshift-pipelines/pipelines-nop-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Pipelines Nop" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Nop" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Nop" \
      io.k8s.description="Red Hat OpenShift Pipelines Nop" \
      io.openshift.tags="pipelines,tekton,openshift" \
      vendor="Red Hat, Inc." \
      distribution-scope="public" \
      url="https://access.redhat.com/containers/#/registry.access.redhat.com/ubi9-minimal/images/9.4-1227.1725849298" \
      release="1227.1725849298"

USER 65532

ENTRYPOINT ["/ko-app/nop"]
