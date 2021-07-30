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

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.runtime.ActorManager;
import com.ibm.research.kar.runtime.KarConfig;

@Path("actor")
public class ActorRuntimeResource {

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		if (ActorManager.getActor(type, id) != null) {
			// Already exists; nothing to do.
			return Response.status(Response.Status.OK).build();
		}

		// Allocate a new actor instance
		ActorInstance actorObj = ActorManager.createActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Not found: " + type + " actor " + id).build();
		}

		// Call the optional activate method
		try {
			MethodHandle activate = ActorManager.getActorActivateMethod(type);
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
		ActorInstance actorObj = ActorManager.getActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).entity("Not found: " + type + " actor " + id).build();
		}

		// Call the optional deactivate method
		MethodHandle deactivate = ActorManager.getActorDeactivateMethod(type);
		if (deactivate != null) {
			try {
				deactivate.invoke(actorObj);
			} catch (Throwable t) {
				return Response.status(Response.Status.BAD_REQUEST).entity(t.toString()).build();
			}
		}

		// Actually remove the instance
		ActorManager.deleteActor(type, id);
		return Response.status(Response.Status.OK).build();
	}

	@HEAD
	@Path("{type}")
	public Response checkActorType(@PathParam("type") String type) {
		Status status = ActorManager.hasActorType(type) ? Response.Status.OK : Response.Status.NOT_FOUND;
		return Response.status(status).build();
	}

	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(Kar.KAR_ACTOR_JSON)
	public Response invokeActorMethod(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("sessionid") String sessionid, @PathParam("path") String path, JsonArray args) {

		ActorInstance actorObj = ActorManager.getActor(type, id);
		if (actorObj == null) {
			return Response.status(Response.Status.NOT_FOUND).type(MediaType.TEXT_PLAIN).entity("Actor instance not found: " + type + "[" + id +"]").build();
		}

		MethodHandle actorMethod = ActorManager.getActorMethod(type, path, args.size());
		if (actorMethod == null) {
			return Response.status(Response.Status.NOT_FOUND).type(MediaType.TEXT_PLAIN).entity("Method not found: " + type + "." + path + " with " + args.size() + " arguments").build();
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
				return Response.status(Response.Status.OK).type(Kar.KAR_ACTOR_JSON).entity(ro).build();
			}
		} catch (Throwable t) {
			if (KarConfig.SHORTEN_ACTOR_STACKTRACES) {
				// Elide all of the implementation details above us in the backtrace
				StackTraceElement [] fullBackTrace = t.getStackTrace();
				for (int i=0; i<fullBackTrace.length; i++) {
					if (fullBackTrace[i].getClassName().equals(ActorRuntimeResource.class.getName()) && fullBackTrace[i].getMethodName().equals("invokeActorMethod")) {
						StackTraceElement[] reducedBackTrace = new StackTraceElement[i+1];
						System.arraycopy(fullBackTrace, 0, reducedBackTrace, 0, i+1);
						t.setStackTrace(reducedBackTrace);
						break;
					}
				}
			}
			JsonObjectBuilder ro = Json.createObjectBuilder();
			ro.add("error", true);
			ro.add("message", t.toString());
			StringWriter sw = new StringWriter();
			PrintWriter pw = new PrintWriter(sw);
			t.printStackTrace(pw);
			String backtrace = sw.toString();
			if (backtrace.length() > KarConfig.MAX_STACKTRACE_SIZE) {
				backtrace = backtrace.substring(0, KarConfig.MAX_STACKTRACE_SIZE) + "\n...Backtrace truncated due to message length restrictions\n";
			}
			ro.add("stack", sw.toString());
			return Response.status(Response.Status.OK).type(Kar.KAR_ACTOR_JSON).entity(ro.build()).build();
		}
	}
}
