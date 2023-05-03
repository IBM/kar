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

package com.ibm.research.kar.example.hello;

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

/**
 * A simple hello service that implements two routes,
 * one that consumes/produces JSON and one that consumes/produces
 * plan text.
 */
@Path("/")
public class HelloServices {

  /**
   * A simple greeting service that consumes/produces JSON values
   *
   * @param body The request body
   * @return A greeting message
   */
  @POST
  @Path("helloJson")
  @Consumes(MediaType.APPLICATION_JSON)
  @Produces(MediaType.APPLICATION_JSON)
  public JsonValue sayHelloJson(JsonValue body) {
    String msg = "Hello " + body.asJsonObject().getString("name");
    System.out.println(msg);
    JsonObjectBuilder jb = JsonProvider.provider().createObjectBuilder();
    jb.add("greetings", msg);
    return jb.build();
  }

  /**
   * A simple greeting service that consumes/produces plain text values
   *
   * @param body The request body
   * @return A greeting message
   */
  @POST
  @Path("helloText")
  @Consumes(MediaType.TEXT_PLAIN)
  @Produces(MediaType.TEXT_PLAIN)
  public String sayHelloText(String body) {
    String msg = "Hello " + body;
    System.out.println(msg);
    return msg;
  }

  /**
   * A simple increment service that consumes/produces JSON values
   *
   * @param body The request body
   * @return A greeting message
   */
  @POST
  @Path("increment")
  @Consumes(MediaType.APPLICATION_JSON)
  @Produces(MediaType.APPLICATION_JSON)
  public JsonValue increment(JsonNumber body) {
    return Json.createValue(body.intValue() + 1);
  }
}
