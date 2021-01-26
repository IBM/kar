#!/bin/bash

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

KAR_VERSION=${KAR_VERSION:="unofficial"}

CIDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$CIDIR/.."

cd $ROOTDIR/CORE
mkdir -p build

buildOne() {
    OS=$1
    ARCH=$2
    OUT=build/$OS-$ARCH
    mkdir -p $OUT
    echo "Building cli for $OS $ARCH"
    GOOS=$OS GOARCH=$ARCH go build -ldflags "-X github.com/IBM/kar.git/core/internal/config.Version=$KAR_VERSION" -o $OUT ./...
    if [[ "$OS" == "windows" ]]; then
        zip -j build/kar-$OS-$ARCH.zip $OUT/kar.exe
    elif [[ "$OS" == "darwin" ]]; then
        zip -j build/kar-mac-$ARCH.zip $OUT/kar
    else
        tar --strip-components 2 -czvf build/kar-$OS-$ARCH.tgz $OUT/kar
    fi
}    

buildOne "darwin" "amd64"
buildOne "linux" "amd64"
buildOne "windows" "amd64"
