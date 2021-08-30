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
import javax.ws.rs.core.Response.ResponseBuilder;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.runtime.ActorInvokeResult;
import com.ibm.research.kar.runtime.ActorManager;
import com.ibm.research.kar.runtime.KarResponse;

@Path("actor")
public class ActorRuntimeResource {

	private final static JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());

	private static Response buildResponse(KarResponse kr) {
		ResponseBuilder rb = Response.status(kr.statusCode);
		if (kr.statusCode != KarResponse.NO_CONTENT && kr.contentType != null) {
			rb.type(kr.contentType);
			if (kr.body instanceof ActorInvokeResult) {
				ActorInvokeResult ar = (ActorInvokeResult) kr.body;
				JsonObjectBuilder jb = factory.createObjectBuilder();
				if (ar.error) {
					jb.add("error", true);
					if (ar.message != null) {
						jb.add("message", ar.message);
					}
					if (ar.stack != null) {
						jb.add("stack", ar.stack);
					}
				} else {
					jb.add("value", ar.value == null ? JsonValue.NULL : (JsonValue) ar.value);
				}
				rb.entity(jb.build());
			} else {
				rb.entity(kr.body);
			}
		}
		return rb.build();
	}

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		return buildResponse(ActorManager.activateInstanceIfNotPresent(type, id));
	}

	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		return buildResponse(ActorManager.deactivateInstanceIfPresent(type, id));
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

		// build arguments array for the actual invoke;
		// the actor instance will be injected into args[0] inside ActorManager.invokeActorMethod.
		Object[] actuals = new Object[args.size() + 1];
		for (int i = 0; i < args.size(); i++) {
			actuals[i + 1] = args.get(i);
		}

		return buildResponse(ActorManager.invokeActorMethod(type, id, sessionid, path, actuals));
	}
}
