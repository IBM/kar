package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;
import java.util.logging.Logger;

import javax.inject.Inject;
import javax.json.JsonArray;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.KarRest;
import com.ibm.research.kar.actor.ActorInstance;

@Path("/")
public class ActorRuntimeResource {

	private static Logger logger = Logger.getLogger(ActorRuntimeResource.class.getName());
	private final static String LOG_PREFIX = "ActorRuntimeResource.";

	@Inject
	ActorManager actorManager;

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		logger.info(LOG_PREFIX + "getActor: Checking for actor with id " + id);
		if (actorManager.getActor(type, id) != null) {
			logger.info(LOG_PREFIX + "getActor: Found actor");
			return Response.status(Response.Status.OK).build();
		} else {
			logger.info(LOG_PREFIX + "getActor: No actor found, creating");
			this.actorManager.createActor(type, id);
			return Response.status(Response.Status.CREATED).entity("Created " + type + " actor " + id).build();
		}
	}

	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		logger.info(LOG_PREFIX + "deleteActor: deleting actor");

		actorManager.deleteActor(type, id);
		return Response.status(Response.Status.OK).build();
	}

	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(KarRest.KAR_ACTOR_JSON)
	@Produces(KarRest.KAR_ACTOR_JSON)
	public Response invokeActorMethod(
			@PathParam("type") String type,
			@PathParam("id") String id,
			@PathParam("sessionid") String sessionid,
			@PathParam("path") String path,
			JsonArray args) {

		logger.finer(LOG_PREFIX + "invokeActorMethod: invoking " + type + " actor " + id + " method " + path + " with args " + args);

		ActorInstance actorObj = this.actorManager.getActor(type, id);
		MethodHandle actorMethod = this.actorManager.getActorMethod(type, path);

		if (actorObj == null) {
			// Internal error.  KAR promises that getActor will be called before it invokes a method on the actor.
			logger.warning(LOG_PREFIX+"invokeActorMethod: Actor instance not found for " + type + "<" + id + ">");
			return Response.status(Response.Status.INTERNAL_SERVER_ERROR).build();
		}

		if (actorMethod == null) {
			logger.info(LOG_PREFIX+"invokeActorMethod: Cannot find method " + path);
			return Response.status(Response.Status.NOT_FOUND).build();
		}

		// set the session
		actorObj.setSession(sessionid);

		// build arguments array for method handle invoke
		Object[] actuals = new Object[args.size()+1];
		actuals[0] = actorObj;
		for (int i = 0; i < args.size(); i++) {
			actuals[i + 1] = args.get(i);
		}

		try {
			Object result = actorMethod.invokeWithArguments(actuals);
			if (result == null) {
				result = JsonValue.NULL;
			}
			return Response.status(Response.Status.OK).entity(result).build();
		} catch (Throwable t) {
			// TODO: Revist the response code for Errors raised by actor methods (https://github.ibm.com/solsa/kar/issues/130)
			return Response.status(Response.Status.BAD_REQUEST).build();
		}
	}
}
