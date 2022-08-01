<!--
# Copyright IBM Corporation 2020,2022
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

This example uses KAR's Actor Programming Model to implement
Dijkstra's solution to the Dining Philosophers problem
(https://en.wikipedia.org/wiki/Dining_philosophers_problem).

The Philosophers and their Forks are all actors and interact via actor
method invocations to implement the distributed protocol that ensures
no Philosopher starves.

Philosophers use a tail call pattern to move from one
state to the next and achieve fault tolerance.

A Cafe may contain an arbitrary number of tables of Philosophers. Each
Cafe tracks its occupancy and generates messages when it seats new
tables or when a sated Philosopher leaves.

To run the example locally, first do an `npm install`.
Then in one window start up the server code:
```shell
kar run -app dp -actors ActorTypeA,ActorTypeB,Tester node philosophers.js
```
In a second window, use the `kar` cli to invite some Philosopers to dinner:
```shell
kar invoke -app dp Tester TesterX startTest
```