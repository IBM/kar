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

package com.ibm.research.kar.example.timeout.front;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

import com.ibm.research.kar.Kar;

/**
 * Frontend that uses KAR SDK to invoke backend service
 */
@Path("/")
public class FrontendServices {

  /**
   * Initiate a test scenario
   *
   * @param body The request body
   * @return A greeting message
   */
  @POST
  @Path("runTest")
  @Consumes(MediaType.APPLICATION_JSON)
  @Produces(MediaType.APPLICATION_JSON)
  public JsonValue runTest(JsonValue body) {
    int count = body.asJsonObject().getInt("count");
    int delay = body.asJsonObject().getInt("delay");
    System.out.println("Initiating test of "+count+" iterations with delay "+delay);
    for (int i=1; i<= count; i++) {
      JsonObjectBuilder jb = Json.createObjectBuilder();
      jb.add("delay", delay);
      jb.add("data", i);
      System.out.println("Initiating request "+i);
      JsonObject response = (JsonObject)Kar.Services.call("backend", "echo", jb.build());
      if (response.getInt("payload") != -i) {
        System.out.println("Error: unexpected response payload "+response.getInt("payload"));
        return JsonValue.FALSE;
      }
      System.out.println("Successfully computed request "+i);
    }

    return JsonValue.TRUE;
  }
}
