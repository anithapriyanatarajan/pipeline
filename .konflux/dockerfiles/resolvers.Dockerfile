ARG GO_BUILDER=brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.24
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:bb08f2300cb8d12a7eb91dddf28ea63692b3ec99e7f0fa71a1b300f2756ea829

FROM $RUNTIME as dependency-builder

COPY dependencies/tini dependencies/tini
WORKDIR /dependencies/tini
RUN microdnf update && microdnf install -y cmake gcc
ENV ENV CFLAGS="-DPR_SET_CHILD_SUBREAPER=36 -DPR_GET_CHILD_SUBREAPER=37"
RUN cmake . && make tini

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/pipeline
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD
ENV GOEXPERIMENT=strictfipsruntime
ENV GODEBUG="http2server=0"
RUN go build -ldflags="-X 'knative.dev/pkg/changeset.rev=$(cat HEAD)'" -mod=vendor -tags disable_gcp,strictfipsruntime -v -o /tmp/resolvers \
    ./cmd/resolvers

FROM $RUNTIME
ARG VERSION=pipeline-1.19

ENV RESOLVERS=/usr/local/bin/resolvers \
    KO_APP=/ko-app \
    KO_DATA_PATH=/kodata

COPY --from=builder /tmp/resolvers /ko-app/resolvers
COPY head ${KO_DATA_PATH}/HEAD

COPY --from=dependency-builder /dependencies/tini/tini /sbin/tini
RUN chmod 0755 /sbin/tini && chown root:root /sbin/tini

LABEL \
      com.redhat.component="openshift-pipelines-resolvers-rhel9-container" \
      name="openshift-pipelines/pipelines-resolvers-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Pipelines Resolvers" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Resolvers" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Resolvers" \
      io.k8s.description="Red Hat OpenShift Pipelines Resolvers" \
      io.openshift.tags="pipelines,tekton,openshift"

RUN microdnf update && microdnf install -y git && microdnf clean all

RUN groupadd -r -g 65532 nonroot && \
    useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT ["/sbin/tini", "--", "/ko-app/resolvers"]

