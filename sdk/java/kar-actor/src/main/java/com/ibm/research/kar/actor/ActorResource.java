package com.ibm.research.kar.actor;


import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
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

	@Inject
	ActorManager actorManager;

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {

		System.out.println("ActorResource.getActor: Checking for actor with id " + id);
		if (actorManager.getActor(type, id) != null) {

			System.out.println("ActorResource.getActor: Found actor");
			return Response.status(Response.Status.OK).build();
		} else {

			System.out.println("ActorResource.getActor: No actor found");
			return Response.status(Response.Status.INTERNAL_SERVER_ERROR ).build();
		}
	}

	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		System.out.println("ActorResource.deleteActor: deleting actor");

		actorManager.deleteActor(type, id);
		return Response.status(Response.Status.OK).build();
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

		System.out.println("ActorResource.invokeActorMethod: invoking actor " + type + ":" + id + " method " + path + " with params " + params);

		ActorReference actorRef = actorManager.getActor(type, id);
		Object actor = actorRef.getActorInstance();
		Method method = actorRef.getRemoteMethods().get(path);

		if (method != null) {
			try {
				Object result = method.invoke(actor);
				return Response
						.status(Response.Status.OK)
						.entity(result)
						.build();
			} catch (SecurityException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
				return this.returnServerError();
			} catch (IllegalAccessException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
				return this.returnServerError();
			} catch (IllegalArgumentException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
				return this.returnServerError();
			} catch (InvocationTargetException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
				return this.returnServerError();
			} 
		}  else {
			return Response
					.status(Response.Status.NOT_FOUND)
					.build();
		}

	}

	private Response returnServerError() {
		return Response
				.status(Response.Status.INTERNAL_SERVER_ERROR)
				.build();
	}
}
