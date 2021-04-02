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

package com.ibm.research.kar.example.timeout.backend;

import javax.json.Json;
import javax.json.JsonNumber;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class SlowAdder extends ActorSkeleton {

  @Remote
  public JsonNumber add(JsonNumber value, JsonNumber delay) {
    int t = delay.intValue();
    System.out.println("Entered add; sleeping "+t+" seconds");
    try {
      Thread.sleep(t * 1000);
    } catch (InterruptedException e) {
      e.printStackTrace();
    }

    System.out.println("Awake: returning "+value.intValue()+1);
    return Json.createValue(value.intValue()+1);
  }

}
