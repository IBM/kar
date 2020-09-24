package com.ibm.research.kar.actor.runtime;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

@Path("system")
public class StatusReporter {

  @GET
	@Path("health")
	@Produces(MediaType.TEXT_PLAIN)
	public Response healthCheck() {
    return Response.status(Response.Status.OK).entity("Peachy Keen!").build();
  }
}
