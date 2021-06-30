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

# TBD: Add more details
This example implements a basic money transfer application between two accounts. It consists of two actors Account and Transaction.

To run the example locally, first do an `npm install`.
Then in one window start up the server code:
```shell
kar run -app money-transfer -actors Account,Transaction node bank_server.js
```
In a second window, use the `kar` cli to initiate a money transfer:
```shell
kar invoke -app money-transfer node bank_client.js
```
