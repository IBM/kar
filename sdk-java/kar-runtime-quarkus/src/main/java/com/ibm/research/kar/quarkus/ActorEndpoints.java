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

import javax.json.JsonArray;
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

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.runtime.ActorManager;

@Path("/kar/impl/v1/actor")
public class ActorEndpoints {

	@GET
	@Path("/{type}/{id}")
	@Produces(MediaType.TEXT_PLAIN)
	public Response getActor(String type, String id) {
		return ActorManager.activateInstanceIfNotPresent(type, id);
	}

	@DELETE
	@Path("/{type}/{id}")
	public Response deleteActor(String type, String id) {
		return ActorManager.deactivateInstanceIfPresent(type, id);
	}

  @HEAD
	@Path("/{type}")
	public Response checkActorType(String type) {
		Status status = ActorManager.hasActorType(type) ? Response.Status.OK : Response.Status.NOT_FOUND;
		return Response.status(status).build();
	}

	@POST
	@Path("/{type}/{id}/{sessionid}/{path}")
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(Kar.KAR_ACTOR_JSON)
	public Response invokeActorMethod(String type, String id, String sessionid, String path, JsonArray args) {
		return ActorManager.invokeActorMethod(type, id, sessionid, path, args);
	}
}
