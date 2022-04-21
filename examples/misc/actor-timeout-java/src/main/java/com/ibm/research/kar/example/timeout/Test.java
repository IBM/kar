/*
 * Copyright IBM Corporation 2020,2022
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
import static com.ibm.research.kar.Kar.Actors.rootCall;
import static com.ibm.research.kar.Kar.Actors.tell;

import java.time.Duration;
import java.time.Instant;
import java.time.temporal.ChronoUnit;

import static com.ibm.research.kar.Kar.Actors.ref;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonString;

import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.Reminder;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

import com.ibm.research.kar.Kar;

@Actor
public class Test extends ActorSkeleton {

  @Remote public void A() {
    System.out.println("Entering method A");
    call(this, this, "B"); // synchronous call to self within the same session -> OK
    System.out.println("Exiting method A");
  }

  @Remote public void B() {
    System.out.println("Entering method B");
    rootCall(this, "A"); // synchronous call to self in a new session -> deadlock
    System.out.println("Exiting method B");
  }

  @Remote public void asyncA() {
    tell(this, "A");
  }

  @Remote public void externalA(JsonString target) {
    ActorRef other = ref("Test", target.getString());
    rootCall(other, "A");
  }

  @Remote public void echo(JsonString msg, JsonNumber count) {
    int n = count.intValue();
    for (int i=0; i<n; i++) {
      System.out.println(getId() + " says "+ msg.getString());
    }
  }

  @Remote public Object incrTailCall(JsonNumber v, JsonNumber toGo) {
    System.out.println("incrTailCall: "+v+" "+toGo);
    if (toGo.intValue() == 0) {
      return v;
    } else {
      return new Kar.Actors.TailCall(this, "incrTailCall", Json.createValue(v.intValue() + 1), Json.createValue(toGo.intValue() - 1));
    }
  }

  @Remote public void schedule(JsonString a, JsonString b) {
    Kar.Actors.Reminders.schedule(this, "echo", "12345", Instant.now().plus(2, ChronoUnit.MINUTES), Duration.ofMillis(1000), Json.createValue("hello"), Json.createValue(3));
    Reminder[] reminders = Kar.Actors.Reminders.get(this, "12345");
    if ( reminders != null && reminders.length > 0) {
        System.out.println("Reminder registered with arguments:"+reminders[0].getArgument(0) + " " + reminders[0].getArgument(1));
    } else {
        if ( reminders == null) {
            System.out.println("reminders not defined (null) ");
        } else {
            System.out.println("reminders.get() returned empty array for id:"+"12345");
        }
    }
  }
}
