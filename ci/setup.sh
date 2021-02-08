#
# Copyright IBM Corporation 2020,2021
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
#

set -eu

HELM_VERSION=v3.3.4
KIND_VERSION=v0.10.0
KUBECTL_VERSION=v1.18.8

# Download and install command line tools
pushd /tmp
  # kubectl
  echo 'installing kubectl'
  curl -Lo ./kubectl https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl
  chmod +x kubectl
  sudo cp kubectl /usr/local/bin/kubectl

  # kind
  echo 'installing kind'
  curl -Lo ./kind https://github.com/kubernetes-sigs/kind/releases/download/$KIND_VERSION/kind-linux-amd64
  chmod +x kind
  sudo cp kind /usr/local/bin/kind

  # helm3
  echo 'installing helm 3'
  curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get-helm-3 > get-helm-3.sh && chmod +x get-helm-3.sh && ./get-helm-3.sh --version $HELM_VERSION

popd
