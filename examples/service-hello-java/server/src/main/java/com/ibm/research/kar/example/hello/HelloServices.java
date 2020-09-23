package com.ibm.research.kar.example.hello;

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
}
