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

import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.json.spi.JsonProvider;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

/**
 * A simple backend service that provides an slow echo service. It receives a
 * request, waits for the specified number of seconds, then echos the response
 */
@Path("/")
public class BackendServices {

  /**
   * A simple echo service that waits, then returns a response that is a
   * deterministic value computed from its inputs
   *
   * @param body The request body
   * @return the c
   */
  @POST
  @Path("echo")
  @Consumes(MediaType.APPLICATION_JSON)
  @Produces(MediaType.APPLICATION_JSON)
  public JsonValue echo(JsonValue body) {
    int delay = body.asJsonObject().getInt("delay");
    int data = body.asJsonObject().getInt("data");
    System.out.println("Received data " + data + "; now sleeping for " + delay + " seconds.");
    try {
      Thread.sleep(delay * 1000);
    } catch (InterruptedException e) {
      e.printStackTrace();
    }
    System.out.println("Awake; returning "+(-data));
    JsonObjectBuilder jb = JsonProvider.provider().createObjectBuilder();
    jb.add("payload", -data);
    return jb.build();
  }
}
