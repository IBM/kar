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

ARG KAR_BASE=localhost:5000/kar/kar-sidecar

FROM $KAR_BASE as kar

FROM node:12-alpine

# Use non-root user "node" defined by node base image
RUN mkdir /kar && chown node /kar && chgrp node /kar && chmod g+s /kar
USER node

# Create exepcted /kar subdirs, will be owned by 'node'
RUN mkdir /kar/bin && mkdir /kar/app

# Copy in kar cli to enable "sidecar-in-container" mode
COPY --from=kar /kar/bin/kar /kar/bin/kar

# Inject wrapper script that detects local vs. sidecar mode
COPY runner /kar/bin/runner
CMD /kar/bin/runner
