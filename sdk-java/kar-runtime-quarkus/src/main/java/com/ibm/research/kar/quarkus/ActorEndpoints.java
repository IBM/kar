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

package com.ibm.research.kar.quarkus;

import java.lang.invoke.MethodHandle;
import java.util.Map;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonObjectBuilder;
import javax.json.JsonBuilderFactory;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.HEAD;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.Kar.Actors.TailCall;
import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.runtime.ActorManager;
import com.ibm.research.kar.runtime.ActorType;
import com.ibm.research.kar.runtime.KarHttpConstants;

import org.jboss.resteasy.reactive.RestQuery;

import io.smallrye.mutiny.Uni;

@Path("/kar/impl/v1/actor")
public class ActorEndpoints implements KarHttpConstants {

	private final static JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());

	/**
	 * Activate an actor instance if it is not already in memory. If the specified
	 * instance is in memory already, simply return success. If the specified
	 * instance it not in memory already, allocate a language-level instance for it
	 * and, if provided by the ActorType, invoke the optional activate method.
	 *
	 * @param type The type of the actor instance to be activated
	 * @param id   The id of the actor instance to be activated
	 * @return A Uni that will indicate success (200, 201) or an error condition (400, 404)
	 */
	@GET
	@Path("/{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Uni<Response> getActor(String type, String id, @RestQuery String session) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance != null) {
			return Uni.createFrom().item(Response.ok().build());
		}

		ActorType actorType = ActorManager.getActorType(type);
		if (actorType == null) {
			Response resp = Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		actorInstance = ActorManager.allocateFreshInstance(actorType, id);
		if (actorInstance == null) {
			Response resp = Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity("Unable to allocate instance: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		// Call the optional activate method
		Response success = Response.status(CREATED).type(TEXT_PLAIN).entity("Created " + type + " actor " + id).build();
		MethodHandle activate = actorType.getActivateMethod();
		if (activate != null) {
			try {
				actorInstance.setSession(session);
				final ActorInstance capturedActorInstance = actorInstance;
				Object result = activate.invokeWithArguments(actorInstance);
				if (result instanceof Uni<?>) {
					return ((Uni<?>)result).chain(res -> {
						capturedActorInstance.setSession(null);
						return Uni.createFrom().item(success);
					});
				} else {
					actorInstance.setSession(null);
				}
			} catch (Throwable t) {
				actorInstance.setSession(null);
				Response failure = Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity(t.toString()).build();
				return Uni.createFrom().item(failure);
			}
		}

		return Uni.createFrom().item(success);
	}

	/**
	 * Deactivate an actor instance. If the ActorType has a deactivate method, it
	 * will be invoked on the instance. The actor instance will be removed from the
	 * in-memory state of the runtime.
	 *
	 * @param type The type of the actor instance to be deactivated
	 * @param id   The id of the actor instance to be deactivated
	 * @return A Uni that will indeicate success (200) or an error condition (400, 404)
	 */
	@DELETE
	@Path("/{type}/{id}")
	public Uni<Response> deleteActor(String type, String id) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance == null) {
			Response resp = Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		// Call the optional deactivate method
		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle deactivateMethod = actorType.getDeactivateMethod();
		if (deactivateMethod != null) {
			try {
				Object result = deactivateMethod.invokeWithArguments(actorInstance);
				if (result instanceof Uni<?>) {
					return ((Uni<?>)result).chain(() -> {
						ActorManager.removeInstanceIfPresent(type, id);
						return Uni.createFrom().item(Response.ok().build());
					});
				}
			} catch (Throwable t) {
				Response resp = Response.status(BAD_REQUEST).type(TEXT_PLAIN).entity(t.toString()).build();
				return Uni.createFrom().item(resp);
			}
		}

		ActorManager.removeInstanceIfPresent(type, id);
		return Uni.createFrom().item(Response.ok().build());
	}

  @HEAD
	@Path("/{type}")
	public Response checkActorType(String type) {
		Status status = ActorManager.hasActorType(type) ? Response.Status.OK : Response.Status.NOT_FOUND;
		return Response.status(status).build();
	}

	/**
	 * Invoke an actor method
	 *
	 * @param type The type of the actor
	 * @param id The id of the target instancr
	 * @param session The session in which the method is being invoked
	 * @param path The method to invoke
	 * @param args The arguments to the method
	 * @return a Uni representing the result of the method invocation
	 */
	@POST
	@Path("/{type}/{id}/{path}")
	@Consumes(KAR_ACTOR_JSON)
	@Produces(KAR_ACTOR_JSON)
	public Uni<Response> invokeActorMethod(String type, String id, @RestQuery String session, String path, JsonArray args) {
		ActorInstance actorObj = ActorManager.getInstanceIfPresent(type, id);
		if (actorObj == null) {
			Response resp = Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Actor instance not found: " + type + "[" + id +"]").build();
			return Uni.createFrom().item(resp);
		}

		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle actorMethod = actorType.getRemoteMethods().get(path + ":" + args.size());
		if (actorMethod == null) {
			Response resp = Response.status(NOT_FOUND).type(TEXT_PLAIN).entity("Method not found: " + type + "." + path + " with " + args.size() + " arguments").build();
			return Uni.createFrom().item(resp);
		}

		// Construct actual argument arrays for the invoke
		Object[] actuals = new Object[args.size() + 1];
		actuals[0] = actorObj;
		for (int i = 0; i < args.size(); i++) {
			actuals[i + 1] = args.get(i);
		}

		String priorSession = actorObj.getSession();
		try {
			actorObj.setSession(session);
			Object result = actorMethod.invokeWithArguments(actuals);
			if (result == null || actorMethod.type().returnType().equals(Void.TYPE)) {
				actorObj.setSession(priorSession);
				return Uni.createFrom().item(Response.status(NO_CONTENT).build());
			} else if (result instanceof Uni<?>) {
				return ((Uni<?>)result)
					.chain(res -> {
						actorObj.setSession(priorSession);
						if (res == null) {
							return Uni.createFrom().item(Response.status(NO_CONTENT).build());
						} else {
							JsonObject response = encodeInvocationResult(res);
							Response resp = Response.ok().type(KAR_ACTOR_JSON).entity(response.toString()).build();
							return Uni.createFrom().item(resp);
						}
					})
					.onFailure().recoverWithItem(t -> {
						actorObj.setSession(priorSession);
						return encodeInvocationError(t);
					});
			} else {
				actorObj.setSession(priorSession);
				JsonObject response = encodeInvocationResult(result);
				Response resp = Response.ok().type(KAR_ACTOR_JSON).entity(response.toString()).build();
				return Uni.createFrom().item(resp);
			}
		} catch (Throwable t) {
			actorObj.setSession(priorSession);
			return Uni.createFrom().item(encodeInvocationError(t));
		}
	}

	private static JsonObject encodeInvocationResult(Object result) {
		JsonObjectBuilder jb = factory.createObjectBuilder();
		if (result instanceof TailCall) {
			TailCall cr = (TailCall)result;
			JsonObjectBuilder crb = factory.createObjectBuilder();
			if (cr.service != null) {
				crb.add("payload", cr.args[0].toString());
				crb.add("serviceName", cr.service);
				crb.add("method", "POST");
			} else {
				JsonArrayBuilder argb = factory.createArrayBuilder();
				for (JsonValue arg: cr.args) {
					argb.add(arg);
				}
				crb.add("payload", argb.build().toString());
				crb.add("actorType", cr.actor.getType());
				crb.add("actorId", cr.actor.getId());
			}
			crb.add("path", "/"+cr.path);
			if (cr.releaseLock) {
				crb.add("releaseLock", "true");
			}
			jb.add("tailCall", true);
			jb.add("value", crb.build());
		} else {
			jb.add("value", (JsonValue)result);
		}
		return jb.build();
	}

	private static Response encodeInvocationError(Throwable t) {
		JsonObjectBuilder jb = factory.createObjectBuilder();
		jb.add("error", true);
		if (t.getMessage() != null) {
			jb.add("message", t.getMessage());
		}
		jb.add("stack", ActorManager.stacktraceToString(t, ActorEndpoints.class.getName(), "invokeActorMethod"));
		return Response.ok().type(KAR_ACTOR_JSON).entity(jb.build().toString()).build();
	}
}
