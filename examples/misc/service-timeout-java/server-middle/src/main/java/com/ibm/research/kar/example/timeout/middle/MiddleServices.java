/*
 * Copyright IBM Corporation 2020,2023
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

package com.ibm.research.kar.example.timeout.middle;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.json.spi.JsonProvider;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.ActorRef;

/**
 * A simple middle tier service that invokes two backend
 * actors and combines their responses to make its own response.
 */
@Path("/")
public class MiddleServices {

  /**
   * A simple middle end service that invokes two
   * backend actor instances with a configurable
   * think time between each step and returns a combined result
   * as its result.
   *
   * @param body The request body
   * @return the c
   */
  @POST
  @Path("doubler")
  @Consumes(MediaType.APPLICATION_JSON)
  @Produces(MediaType.APPLICATION_JSON)
  public JsonValue doubler(JsonValue body) {
    int delay = body.asJsonObject().getInt("delay");
    int data = body.asJsonObject().getInt("data");
    System.out.println("Received data " + data + "; now sleeping for " + delay + " seconds.");
    try {
      Thread.sleep(delay * 1000);
    } catch (InterruptedException e) {
      e.printStackTrace();
    }

    System.out.println("Awake; invoking first actor");
    ActorRef myBackend = Kar.Actors.ref("SlowAdder", "Singleton");
    JsonValue firstPart = Kar.Actors.rootCall(myBackend, "add", Json.createValue(data), Json.createValue(delay));

    System.out.println("Received response " + firstPart + "; now sleeping for " + delay + " seconds.");
    try {
      Thread.sleep(delay * 1000);
    } catch (InterruptedException e) {
      e.printStackTrace();
    }

    System.out.println("Awake; invoking second actor");
    JsonValue secondPart = Kar.Actors.rootCall(myBackend, "add", Json.createValue(data), Json.createValue(delay));
    System.out.println("Received response " + secondPart + "; now sleeping for " + delay + " seconds.");
    try {
      Thread.sleep(delay * 1000);
    } catch (InterruptedException e) {
      e.printStackTrace();
    }

    System.out.println("Awake; returning result");
    JsonObjectBuilder jb = JsonProvider.provider().createObjectBuilder();
    jb.add("payload", ((JsonNumber)firstPart).intValue() + ((JsonNumber)secondPart).intValue());
    return jb.build();
  }
}
