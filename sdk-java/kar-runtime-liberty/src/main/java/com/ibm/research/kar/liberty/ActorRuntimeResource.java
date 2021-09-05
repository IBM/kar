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

package com.ibm.research.kar.liberty;

import java.lang.invoke.MethodHandle;
import java.util.Map;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonBuilderFactory;
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

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.runtime.ActorManager;
import com.ibm.research.kar.runtime.ActorType;
import com.ibm.research.kar.runtime.KarHttpConstants;

@Path("actor")
public class ActorRuntimeResource implements KarHttpConstants {

	private final static JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());

	/**
	 * Activate an actor instance if it is not already in memory. If the specified
	 * instance is in memory already, simply return success. If the specified
	 * instance it not in memory already, allocate a language-level instance for it
	 * and, if provided by the ActorType, invoke the optional activate method.
	 *
	 * @param type The type of the actor instance to be activated
	 * @param id   The id of the actor instance to be activated
	 * @return A Response indicating success (200, 201) or an error condition (400, 404)
	 */
	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance != null) {
			return Response.ok().build();
		}

		ActorType actorType = ActorManager.getActorType(type);
		if (actorType == null) {
			return Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
		}

		actorInstance = ActorManager.allocateFreshInstance(actorType, id);
		if (actorInstance == null) {
			return Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity("Unable to allocate instance: " + type + " actor " + id).build();
		}

		// Call the optional activate method
		MethodHandle activate = actorType.getActivateMethod();
		if (activate != null) {
			try {
				activate.invoke(actorInstance);
			}  catch (Throwable t) {
				return Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity(t.toString()).build();
			}
		}

		return Response.status(CREATED).type(TEXT_PLAIN).entity("Created " + type + " actor " + id).build();
	}

	/**
	 * Deactivate an actor instance. If the ActorType has a deactivate method, it
	 * will be invoked on the instance. The actor instance will be removed from the
	 * in-memory state of the runtime.
	 *
	 * @param type The type of the actor instance to be deactivated
	 * @param id   The id of the actor instance to be deactivated
	 * @return A Response indicating success (200) or an error condition (400, 404)
	 */
	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance == null) {
			return Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
		}

		// Call the optional deactivate method
		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle deactivateMethod = actorType.getDeactivateMethod();
		if (deactivateMethod != null) {
			try {
				deactivateMethod.invoke(actorInstance);
			} catch (Throwable t) {
				return Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity(t.toString()).build();
			}
		}

		// Actually remove the instance from memory
		ActorManager.removeInstanceIfPresent(type, id);

		return (Response.ok().build());
	}

	@HEAD
	@Path("{type}")
	public Response checkActorType(@PathParam("type") String type) {
		Status status = ActorManager.hasActorType(type) ? Response.Status.OK : Response.Status.NOT_FOUND;
		return Response.status(status).build();
	}

	/**
	 * Invoke an actor method
	 *
	 * @param type The type of the actor
	 * @param id The id of the target instancr
	 * @param sessionid The session in which the method is being invoked
	 * @param path The method to invoke
	 * @param args The arguments to the method
	 * @return a Response containing the result of the method invocation
	 */
	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(Kar.KAR_ACTOR_JSON)
	public Response invokeActorMethod(@PathParam("type") String type, @PathParam("id") String id, @PathParam("sessionid") String sessionid, @PathParam("path") String path, JsonArray args) {
		ActorInstance actorObj = ActorManager.getInstanceIfPresent(type, id);
		if (actorObj == null) {
			return Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Actor instance not found: " + type + "[" + id +"]").build();
		}

		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle actorMethod = actorType.getRemoteMethods().get(path + ":" + args.size());
		if (actorMethod == null) {
			return Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Method not found: " + type + "." + path + " with " + args.size() + " arguments").build();
		}

		// set the session
		actorObj.setSession(sessionid);

		// Construct actual argument arrays for the invoke
		Object[] actuals = new Object[args.size() + 1];
		actuals[0] = actorObj;
		for (int i = 0; i < args.size(); i++) {
			actuals[i + 1] = args.get(i);
		}

		try {
			Object result = actorMethod.invokeWithArguments(actuals);
			if (result == null || actorMethod.type().returnType().equals(Void.TYPE)) {
				return Response.status(NO_CONTENT).build();
			} else {
				JsonObjectBuilder jb = factory.createObjectBuilder();
				jb.add("value", (JsonValue)result);
				return Response.ok().type(KAR_ACTOR_JSON).entity(jb.build().toString()).build();
			}
		} catch (Throwable t) {
			JsonObjectBuilder jb = factory.createObjectBuilder();
			jb.add("error", true);
			if (t.getMessage() != null) {
				jb.add("message", t.getMessage());
			}
			jb.add("stack", ActorManager.stacktraceToString(t, ActorRuntimeResource.class.getName(), "invokeActorMethod"));
			return Response.ok().type(KAR_ACTOR_JSON).entity(jb.build().toString()).build();
		}
	}
}
