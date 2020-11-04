package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;
import java.util.logging.Logger;

import java.io.StringWriter;
import java.io.PrintWriter;

import javax.inject.Inject;
import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.HEAD;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.KarRest;
import com.ibm.research.kar.actor.ActorInstance;

@Path("actor")
public class ActorRuntimeResource {

	private static Logger logger = Logger.getLogger(ActorRuntimeResource.class.getName());
	private final static String LOG_PREFIX = "ActorRuntimeResource.";

	@Inject
	ActorManager actorManager;

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		if (actorManager.getActor(type, id) != null) {
			// Already exists; nothing to do.
			return Response.status(Response.Status.OK).build();
		}

		// Allocate a new actor instance
		ActorInstance actorObj = this.actorManager.createActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Not found: " + type + " actor " + id).build();
		}

		// Call the optional activate method
		try {
			MethodHandle activate = this.actorManager.getActorActivateMethod(type);
			if (activate != null) {
				activate.invoke(actorObj);
			}
			return Response.status(Response.Status.CREATED).entity("Created " + type + " actor " + id).build();
		} catch (Throwable t) {
			return Response.status(Response.Status.BAD_REQUEST).entity(t.toString()).build();
		}
	}

	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		ActorInstance actorObj = this.actorManager.getActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Not found: " + type + " actor " + id).build();
		}

		// Call the optional deactivate method
		MethodHandle deactivate = this.actorManager.getActorDeactivateMethod(type);
		if (deactivate != null) {
			try {
				deactivate.invoke(actorObj);
			} catch (Throwable t) {
				return Response.status(Response.Status.BAD_REQUEST).entity(t.toString()).build();
			}
		}

		// Actually remove the instance
		actorManager.deleteActor(type, id);
		return Response.status(Response.Status.OK).build();
	}

	@HEAD
	@Path("{type}")
	public Response checkActorType(@PathParam("type") String type) {
		Status status = this.actorManager.hasActorType(type) ? Response.Status.OK : Response.Status.NOT_FOUND;
		return Response.status(status).build();
	}

	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(KarRest.KAR_ACTOR_JSON)
	@Produces(KarRest.KAR_ACTOR_JSON)
	public Response invokeActorMethod(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("sessionid") String sessionid, @PathParam("path") String path, JsonArray args) {

		ActorInstance actorObj = this.actorManager.getActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Actor instance not found " + type + "actor" + id).build();
		}

		MethodHandle actorMethod = this.actorManager.getActorMethod(type, path, args.size());
		if (actorMethod == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Cannot find method " + path).build();
		}

		// set the session
		actorObj.setSession(sessionid);

		// build arguments array for method handle invoke
		Object[] actuals = new Object[args.size() + 1];
		actuals[0] = actorObj;
		for (int i = 0; i < args.size(); i++) {
			actuals[i + 1] = args.get(i);
		}

		try {
			Object result = actorMethod.invokeWithArguments(actuals);
			if (result == null && actorMethod.type().returnType().equals(Void.TYPE)) {
				return Response.status(Response.Status.NO_CONTENT).build();
			} else {
				JsonValue jv = result != null ? (JsonValue)result : JsonValue.NULL;
				JsonObject ro = Json.createObjectBuilder().add("value", jv).build();
				return Response.status(Response.Status.OK).type(KarRest.KAR_ACTOR_JSON).entity(ro).build();
			}
		} catch (Throwable t) {
			JsonObjectBuilder ro = Json.createObjectBuilder();
			ro.add("error", true);
			ro.add("message", t.toString());
			StringWriter sw = new StringWriter();
			PrintWriter pw = new PrintWriter(sw);
			t.printStackTrace(pw);
			ro.add("stack", sw.toString());
			return Response.status(Response.Status.OK).type(KarRest.KAR_ACTOR_JSON).entity(ro.build()).build();
		}
	}
}
