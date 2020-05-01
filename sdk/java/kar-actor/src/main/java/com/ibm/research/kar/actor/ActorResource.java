package com.ibm.research.kar.actor;

import javax.enterprise.context.ApplicationScoped;
import javax.json.JsonObject;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

@Path("/actor")
@ApplicationScoped
public class ActorResource {

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public String getActor(@PathParam("type") String type, @PathParam("id") String id) {
		System.out.println("getting actor");
		
		return "Hello friend";
	}
	
	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		System.out.println("deleting actor");
		return null;
	}
	
	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response invokeActorMethod(
			@PathParam("type") String type, 
			@PathParam("id") String id, 
			@PathParam("sessionid") String sessionid, 
			@PathParam("path") String path, 
			JsonObject params) {
		
		System.out.println("invoking actor " + type + ":" + id + " method " + path + " with params " + params);
		
		return Response
				.status(Response.Status.OK)
				.entity("Hello from actor runtime")
				.build();
	}
}
