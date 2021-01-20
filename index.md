<!--
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
-->

## KAR: Kubernetes Application Runtime

The KAR runtime provides a RESTful API to application processes.
You can browse the [swagger specification](https://pages.github.ibm.com/solsa/kar/api/swagger.json) of that API as rendered using:
* [Redoc](https://pages.github.ibm.com/solsa/kar/api/redoc/)
* [Swagger-UI](https://pages.github.ibm.com/solsa/kar/api/swagger-ui/)

Development note: to update the swagger.json, do `make swagger-gen` on the main branch and then commit the updated swagger.json and swagger.yaml files to the gh-pages branch. TODO: Automate this via a GH action.



