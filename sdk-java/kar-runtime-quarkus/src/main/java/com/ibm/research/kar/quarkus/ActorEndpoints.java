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

package com.ibm.research.kar.quarkus;

import java.lang.invoke.MethodHandle;
import java.util.Map;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObjectBuilder;
import javax.json.JsonBuilderFactory;
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

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.runtime.ActorManager;
import com.ibm.research.kar.runtime.ActorType;
import com.ibm.research.kar.runtime.KarResponse;

import org.jboss.resteasy.reactive.common.util.Encode;

import io.smallrye.mutiny.Uni;

@Path("/kar/impl/v1/actor")
public class ActorEndpoints {

	private final static JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());

	@GET
	@Path("/{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Uni<Response> getActor(String type, String id) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance != null) {
			return Uni.createFrom().item(Response.ok().build());
		}

		ActorType actorType = ActorManager.getActorType(type);
		if (actorType == null) {
			Response resp = Response.status(KarResponse.NOT_FOUND).type(KarResponse.TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		actorInstance = ActorManager.allocateFreshInstance(actorType, id);
		if (actorInstance == null) {
			Response resp = Response.status(KarResponse.BAD_REQUEST).type(KarResponse.TEXT_PLAIN).entity("Unable to allocate instance: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		// Call the optional activate method
		try {
			Response success = Response.status(KarResponse.CREATED).type(KarResponse.TEXT_PLAIN).entity("Created " + type + " actor " + id).build();
			MethodHandle activate = actorType.getActivateMethod();
			if (activate != null) {
				Object result = activate.invoke(actorInstance);
				if (result instanceof Uni<?>) {
					return ((Uni<?>)result).chain(() -> Uni.createFrom().item(success));
				}
			}
			return Uni.createFrom().item(success);
		} catch (Throwable t) {
			Response failure = Response.status(KarResponse.BAD_REQUEST).type(KarResponse.TEXT_PLAIN).entity(t.toString()).build();
			return Uni.createFrom().item(failure);
		}
	}

	@DELETE
	@Path("/{type}/{id}")
	public Uni<Response> deleteActor(String type, String id) {
		ActorInstance actorInstance = ActorManager.getInstanceIfPresent(type, id);
		if (actorInstance == null) {
			Response resp = Response.status(KarResponse.NOT_FOUND).type(KarResponse.TEXT_PLAIN).entity("Not found: " + type + " actor " + id).build();
			return Uni.createFrom().item(resp);
		}

		// Call the optional deactivate method
		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle deactivateMethod = actorType.getDeactivateMethod();
		if (deactivateMethod != null) {
			try {
				Object result = deactivateMethod.invoke(actorInstance);
				if (result instanceof Uni<?>) {
					return ((Uni<?>)result).chain(() -> {
						ActorManager.removeInstanceIfPresent(type, id);
						return Uni.createFrom().item(Response.ok().build());
					});
				}
			} catch (Throwable t) {
				Response resp = Response.status(KarResponse.BAD_REQUEST).type(KarResponse.TEXT_PLAIN).entity(t.toString()).build();
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

	@POST
	@Path("/{type}/{id}/{sessionid}/{path}")
	@Consumes(KarResponse.KAR_ACTOR_JSON)
	@Produces(KarResponse.KAR_ACTOR_JSON)
	public Uni<Response> invokeActorMethod(String type, String id, String sessionid, String path, JsonArray args) {
		ActorInstance actorObj = ActorManager.getInstanceIfPresent(type, id);
		if (actorObj == null) {
			Response resp = Response.status(KarResponse.NOT_FOUND).type(KarResponse.TEXT_PLAIN).entity("Actor instance not found: " + type + "[" + id +"]").build();
			return Uni.createFrom().item(resp);
		}

		ActorType actorType = ActorManager.getActorType(type);
		MethodHandle actorMethod = actorType.getRemoteMethods().get(path + ":" + args.size());
		if (actorMethod == null) {
			Response resp = Response.status(KarResponse.NOT_FOUND).type(KarResponse.TEXT_PLAIN).entity("Method not found: " + type + "." + path + " with " + args.size() + " arguments").build();
			return Uni.createFrom().item(resp);
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
				return Uni.createFrom().item(Response.status(KarResponse.NO_CONTENT).build());
			} else if (result instanceof Uni<?>) {
				return ((Uni<?>)result)
					.chain(res -> {
						if (res == null) {
							return Uni.createFrom().item(Response.status(KarResponse.NO_CONTENT).build());
						} else {
							JsonObjectBuilder jb = factory.createObjectBuilder();
							jb.add("value", (JsonValue)res);
							Response resp = Response.ok().type(KarResponse.KAR_ACTOR_JSON).entity(jb.build().toString()).build();
							return Uni.createFrom().item(resp);
						}
					})
					.onFailure().recoverWithItem(t -> encodeInvocationError(t));
			} else {
				JsonObjectBuilder jb = factory.createObjectBuilder();
				jb.add("value", (JsonValue)result);
				Response resp = Response.ok().type(KarResponse.KAR_ACTOR_JSON).entity(jb.build().toString()).build();
				return Uni.createFrom().item(resp);
			}
		} catch (Throwable t) {
			return Uni.createFrom().item(encodeInvocationError(t));
		}
	}

	private static Response encodeInvocationError(Throwable t) {
		JsonObjectBuilder jb = factory.createObjectBuilder();
		jb.add("error", true);
		if (t.getMessage() != null) {
			jb.add("message", t.getMessage());
		}
		jb.add("stack", ActorManager.stacktraceToString(t, ActorEndpoints.class.getName(), "invokeActorMethod"));
		return Response.ok().type(KarResponse.KAR_ACTOR_JSON).entity(jb.build().toString()).build();
	}
}
