#!/usr/bin/env bash

# Copyright 2020 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o nounset
set -o errexit
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")
source "${SCRIPT_ROOT}/../../lib/lib.sh"
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

# Installs an instance of ingress-nginx using the 'stable' Helm chart.
# Configure the cluster to target using the KUBECONFIG environment variable.
# Additional parameters can be configured by overriding the variables below.

# Namespace to deploy into
NAMESPACE="${NAMESPACE:-ingress-nginx}"
if [[ "$IS_OPENSHIFT" == "true" ]] ; then
  # OpenShift needs bind to be in kube-system due to file ownership restrictions
  NAMESPACE="kube-system"
fi
# Release name to use with Helm
RELEASE_NAME="${RELEASE_NAME:-ingress-nginx}"
IMAGE_TAG=""
HELM_CHART=""
INGRESS_WITHOUT_CLASS=""

# Require helm, kubectl and yq available on PATH
check_tool kubectl
check_tool helm
bazel build //hack/bin:yq
bindir="$(bazel info bazel-bin)"
export PATH="${bindir}/hack/bin/:$PATH"

IMAGE_TAG="v1.1.0"
HELM_CHART="4.0.10"

# TODO (irbekrm): is this comment and configuration setting still accurate post-https://github.com/cert-manager/cert-manager/pull/4783?
# v1 NGINX-Ingress by default only watches Ingresses with Ingress class
# defined. When configuring solver block for ACME HTTTP01 challenge on an ACME
# issuer, cert-manager users can currently specify either an Ingress name or a
# class. We also e2e test these two ways of creating Ingresses with
# ingress-shim. For the ingress controller to watch our Ingresses that don't
# have a class, we pass a --watch-ingress-without-class flag
# https://github.com/kubernetes/ingress-nginx/blob/main/charts/ingress-nginx/values.yaml#L64-L67
INGRESS_WITHOUT_CLASS="true"
require_image "k8s.gcr.io/ingress-nginx/controller:${IMAGE_TAG}" "//devel/addon/ingressnginx:bundle"

# Ensure the ingress-nginx namespace exists
kubectl get namespace "${NAMESPACE}" || kubectl create namespace "${NAMESPACE}"

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx

helm repo update

# Upgrade or install nginx-ingress
helm upgrade \
    --install \
    --wait \
    --version "${HELM_CHART}" \
    --namespace "${NAMESPACE}" \
    --set controller.image.digest="" \
    --set controller.image.tag="${IMAGE_TAG}" \
    --set controller.image.pullPolicy=Never \
    --set "controller.service.clusterIP=${SERVICE_IP_PREFIX}.15"\
    --set controller.service.type=ClusterIP \
    --set controller.config.no-tls-redirect-locations="" \
    --set admissionWebhooks.enabled=false \
    --set controller.admissionWebhooks.enabled=false \
    --set controller.watchIngressWithoutClass="${INGRESS_WITHOUT_CLASS}" \
    "$RELEASE_NAME" \
    ingress-nginx/ingress-nginx
