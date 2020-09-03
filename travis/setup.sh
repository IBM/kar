set -eu

HELM_VERSION=v3.2.0
KIND_VERSION=v0.8.1
KUBECTL_VERSION=v1.16.9

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

  # s2i
  echo 'installing s2i'
  # Kludge.  The tar file we download screws up /tmp if we directly untar it; so create a tmp dir to work in.
  mkdir s2i-tmp
  pushd s2i-tmp
  curl -Lo ./s2i.tar.gz https://github.com/openshift/source-to-image/releases/download/v1.3.0/source-to-image-v1.3.0-eed2850f-linux-amd64.tar.gz
  sudo tar xzf  s2i.tar.gz
  sudo chmod +x s2i
  sudo cp s2i /usr/local/bin/s2i
  popd
  sudo rm -rf s2i-tmp

popd
