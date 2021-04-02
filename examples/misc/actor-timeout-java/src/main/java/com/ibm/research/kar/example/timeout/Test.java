/*
 * Copyright IBM Corporation 2020,2021
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.ibm.research.kar.example.timeout;

import static com.ibm.research.kar.Kar.Actors.call;
import static com.ibm.research.kar.Kar.Actors.tell;
import static com.ibm.research.kar.Kar.Actors.ref;

import javax.json.JsonString;

import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Test extends ActorSkeleton {

  @Remote public void A() {
    System.out.println("Entering method A");
    call(this, this, "B"); // synchronous call to self within the same session -> OK
    System.out.println("Exiting method A");
  }

  @Remote public void B() {
    System.out.println("Entering method B");
    call(this, "A"); // synchronous call to self in a new session -> deadlock
    System.out.println("Exiting method B");
  }

  @Remote public void asyncA() {
    tell(this, "A");
  }

  @Remote public void externalA(JsonString target) {
    ActorRef other = ref("Test", target.getString());
    call(other, "A");
  }
}
