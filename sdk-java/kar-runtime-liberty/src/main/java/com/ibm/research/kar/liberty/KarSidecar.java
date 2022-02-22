/*
 * Copyright IBM Corporation 2020,2022
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

import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.HEAD;
import javax.ws.rs.OPTIONS;
import javax.ws.rs.PATCH;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.QueryParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.runtime.KarConfig;

import org.eclipse.microprofile.faulttolerance.Retry;
import org.eclipse.microprofile.faulttolerance.Timeout;
import org.eclipse.microprofile.rest.client.annotation.ClientHeaderParam;
import org.eclipse.microprofile.rest.client.annotation.RegisterProvider;

@Consumes(MediaType.APPLICATION_JSON)
@Produces(MediaType.APPLICATION_JSON)
@Timeout(0)
@Path("kar/v1")
@RegisterProvider(JSONProvider.class)
public interface KarSidecar {

	/*
	 * Services
	 */

	// asynchronous service invocation, returns (202, "OK")
	@DELETE
	@Path("service/{service}/call/{path}")
	@ClientHeaderParam(name = "Pragma", value = "async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response tellDelete(@PathParam("service") String service, @PathParam("path") String path);

	@PATCH
	@Path("service/{service}/call/{path}")
	@ClientHeaderParam(name = "Pragma", value = "async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response tellPatch(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	@POST
	@Path("service/{service}/call/{path}")
	@ClientHeaderParam(name = "Pragma", value = "async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response tellPost(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	@PUT
	@Path("service/{service}/call/{path}")
	@ClientHeaderParam(name = "Pragma", value = "async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response tellPut(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	// synchronous service invocation, returns invocation result
	@DELETE
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callDelete(@PathParam("service") String service, @PathParam("path") String path);

	@GET
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callGet(@PathParam("service") String service, @PathParam("path") String path);

	@HEAD
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callHead(@PathParam("service") String service, @PathParam("path") String path);

	@OPTIONS
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callOptions(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	@PATCH
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callPatch(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	@POST
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callPost(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	@PUT
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response callPut(@PathParam("service") String service, @PathParam("path") String path, JsonValue params);

	// asynchronous service invocation, returns CompletionStage that will contain
	// the eventual invocation result
	@DELETE
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncDelete(@PathParam("service") String service,
			@PathParam("path") String path);

	@GET
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncGet(@PathParam("service") String service, @PathParam("path") String path);

	@HEAD
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncHead(@PathParam("service") String service, @PathParam("path") String path);

	@OPTIONS
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncOptions(@PathParam("service") String service,
			@PathParam("path") String path, JsonValue params);

	@PATCH
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncPatch(@PathParam("service") String service, @PathParam("path") String path,
			JsonValue params);

	@POST
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncPost(@PathParam("service") String service, @PathParam("path") String path,
			JsonValue params);

	@PUT
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public CompletionStage<Response> callAsyncPut(@PathParam("service") String service, @PathParam("path") String path,
			JsonValue params);

	/*
	 * Actors
	 */

	// asynchronous actor invocation, returns (202, "OK")
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@ClientHeaderParam(name = "Pragma", value = "async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorTell(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path,
			JsonArray args);

	// synchronous actor invocation: returns invocation result
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(Kar.KAR_ACTOR_JSON)
	public Response actorCall(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path,
			@QueryParam("session") String session, JsonArray args);

	// synchronous actor invocation: returns invocation result
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(Kar.KAR_ACTOR_JSON)
	@Produces(Kar.KAR_ACTOR_JSON)
	public CompletionStage<Response> actorCallAsync(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("path") String path, @QueryParam("session") String session, JsonArray args);

	//
	// Actor Reminder operations
	//

	@DELETE
	@Path("actor/{type}/{id}/reminders")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorCancelReminders(@PathParam("type") String type, @PathParam("id") String id);

	@DELETE
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorCancelReminder(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("reminderId") String reminderId, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@GET
	@Path("actor/{type}/{id}/reminders")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetReminders(@PathParam("type") String type, @PathParam("id") String id);

	@GET
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetReminder(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("reminderId") String reminderId, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@PUT
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorScheduleReminder(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("reminderId") String reminderId, JsonObject params);

	//
	// Actor state operations
	//

	@GET
	@Path("actor/{type}/{id}/state/{key}/{subkey}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetWithSubkeyState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @PathParam("subkey") String subkey, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@HEAD
	@Path("actor/{type}/{id}/state/{key}/{subkey}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorHeadWithSubkeyState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @PathParam("subkey") String subkey);

	@PUT
	@Path("actor/{type}/{id}/state/{key}/{subkey}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorSetWithSubkeyState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @PathParam("subkey") String subkey, JsonValue params);

	@DELETE
	@Path("actor/{type}/{id}/state/{key}/{subkey}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDeleteWithSubkeyState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @PathParam("subkey") String subkey, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@GET
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@HEAD
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorHeadState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key);

	@PUT
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorSetState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, JsonValue params);

	@POST
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.APPLICATION_JSON)
	public Response actorSubmapOp(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key,
			JsonValue params);

	@DELETE
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDeleteState(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("key") String key, @QueryParam("nilOnAbsent") boolean nilOnAbsent);

	@GET
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetAllState(@PathParam("type") String type, @PathParam("id") String id);

	@POST
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorUpdate(@PathParam("type") String type, @PathParam("id") String id, JsonValue params);

	@DELETE
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDeleteAllState(@PathParam("type") String type, @PathParam("id") String id);

	@DELETE
	@Path("actor/{type}/{id}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDelete(@PathParam("type") String type, @PathParam("id") String id);

	/*
	 * Events
	 */

	@GET
	@Path("actor/{type}/{id}/events")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetAllSubscriptions(@PathParam("type") String type, @PathParam("id") String id);

	@DELETE
	@Path("actor/{type}/{id}/events")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorCancelAllSubscriptions(@PathParam("type") String type, @PathParam("id") String id);

	@GET
	@Path("actor/{type}/{id}/events/{subscriptionId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetSubscription(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("subscriptionId") String subscriptionId);

	@DELETE
	@Path("actor/{type}/{id}/events/{subscriptionId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorCancelSubscription(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("subscriptionId") String subscriptionId);

	@PUT
	@Path("actor/{type}/{id}/events/{subscriptionId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorSubscribe(@PathParam("type") String type, @PathParam("id") String id,
			@PathParam("subscriptionId") String subscriptionId, JsonValue data);

	@PUT
	@Path("event/{topic}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response eventCreateTopic(@PathParam("topic") String topic, JsonValue configuration);

	@DELETE
	@Path("event/{topic}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response eventDeleteTopic(@PathParam("topic") String topic);

	@POST
	@Path("event/{topic}/publish")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response eventPublish(@PathParam("topic") String topic, JsonValue event);

	/*
	 * System
	 */

	@POST
	@Path("system/shutdown")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response shutdown();

	@GET
	@Path("system/information/{component}")
	@ClientHeaderParam(name = "Accepts", value = "application/json")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.APPLICATION_JSON)
	public Response systemInformation(@PathParam("component") String component);
}
