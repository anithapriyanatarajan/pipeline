ARG GO_BUILDER=brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.24
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:bb08f2300cb8d12a7eb91dddf28ea63692b3ec99e7f0fa71a1b300f2756ea829

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/pipeline
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD
ENV GOEXPERIMENT=strictfipsruntime
ENV GODEBUG="http2server=0"
RUN CGO_ENABLED=0 \
    go build -ldflags="-X 'knative.dev/pkg/changeset.rev=$(cat HEAD)'" -mod=vendor -tags disable_gcp,disable_spire,disable_tls,strictfipsruntime -v -o /tmp/entrypoint \
    ./cmd/entrypoint

FROM $RUNTIME
ARG VERSION=pipeline-next

ENV ENTRYPOINT=/usr/local/bin/entrypoint \
    KO_APP=/ko-app \
    KO_DATA_PATH=/kodata

COPY --from=builder /tmp/entrypoint /ko-app/entrypoint
COPY head ${KO_DATA_PATH}/HEAD

LABEL \
      com.redhat.component="openshift-pipelines-entrypoint-rhel9-container" \
      name="openshift-pipelines/pipelines-entrypoint-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Pipelines Entrypoint" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Entrypoint" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Entrypoint" \
      io.k8s.description="Red Hat OpenShift Pipelines Entrypoint" \
      io.openshift.tags="pipelines,tekton,openshift"

RUN groupadd -r -g 65532 nonroot && \
    useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT ["/ko-app/entrypoint"]
