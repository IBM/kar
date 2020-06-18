package com.ibm.research.kar;

import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.QueryParam;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.faulttolerance.Retry;
import org.eclipse.microprofile.faulttolerance.Timeout; 
import org.eclipse.microprofile.rest.client.annotation.ClientHeaderParam;
import org.eclipse.microprofile.rest.client.annotation.RegisterProvider;

import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;

@Consumes(MediaType.APPLICATION_JSON)
@Produces(MediaType.APPLICATION_JSON)
@Timeout(600000)
@Path("kar/v1")
@RegisterProvider(JSONProvider.class)
public interface KarRest extends AutoCloseable {

	public final static String KAR_ACTOR_JSON = "application/kar+json";

	/*
	 * Services
	 */

	// asynchronous service invocation, returns  returns (202, "OK")
	@POST
	@Path("service/{service}/call/{path}")
	@ClientHeaderParam(name="Pragma", value="async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response tell(@PathParam("service") String service, @PathParam("path") String path, JsonValue params) throws ProcessingException;

	// synchronous service invocation, returns invocation result
	@POST
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes({
		MediaType.APPLICATION_JSON,
		MediaType.TEXT_PLAIN,
		MediaType.APPLICATION_XML
	})
	@Produces({
		MediaType.APPLICATION_JSON,
		MediaType.TEXT_PLAIN,
		MediaType.APPLICATION_XML
	})
	public Response call(@PathParam("service") String service, @PathParam("path") String path, JsonValue params) throws ProcessingException;

	// asynchronous service invocation, returns invocation result
	@POST
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes({
		MediaType.APPLICATION_JSON,
		MediaType.TEXT_PLAIN,
		MediaType.APPLICATION_XML
	})
	@Produces({
		MediaType.APPLICATION_JSON,
		MediaType.TEXT_PLAIN,
		MediaType.APPLICATION_XML
	})
	public CompletionStage<Response> callAsync(@PathParam("service") String service, @PathParam("path") String path, JsonValue params) throws ProcessingException;


	/*
	 * Actors
	 */

	// asynchronous actor invocation, returns (202, "OK")
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@ClientHeaderParam(name="Pragma", value="async")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(KAR_ACTOR_JSON)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorTell(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, JsonArray args) throws ProcessingException;

	// synchronous actor invocation: returns invocation result
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(KAR_ACTOR_JSON)
	@Produces(KAR_ACTOR_JSON)
	public JsonValue actorCall(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, @QueryParam("session") String session, JsonArray args) throws ActorMethodNotFoundException;

	// synchronous actor invocation: returns invocation result
	@POST
	@Path("actor/{type}/{id}/call/{path}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Consumes(KAR_ACTOR_JSON)
	@Produces(KAR_ACTOR_JSON)
	public CompletionStage<JsonValue> actorCallAsync(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, @QueryParam("session") String session, JsonArray args) throws ActorMethodNotFoundException;

	//
	// Actor Reminder operations
	//

	@DELETE
	@Path("actor/{type}/{id}/reminders")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorCancelReminders(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;

	@DELETE
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorCancelReminder(@PathParam("type") String type, @PathParam("id") String id, @PathParam("reminderId") String reminderId, @QueryParam("nilOnAbsent") boolean nilOnAbsent) throws ProcessingException;

	@GET
	@Path("actor/{type}/{id}/reminders")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetReminders(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;

	@GET
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetReminder(@PathParam("type") String type, @PathParam("id") String id, @PathParam("reminderId") String reminderId, @QueryParam("nilOnAbsent") boolean nilOnAbsent) throws ProcessingException;

	@POST
	@Path("actor/{type}/{id}/reminders/{reminderId}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorScheduleReminder(@PathParam("type") String type, @PathParam("id") String id, @PathParam("reminderId") String reminderId, JsonObject params) throws ProcessingException;


	//
	// Actor state operations
	//

	@GET
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, @QueryParam("nilOnAbsent") boolean nilOnAbsent) throws ProcessingException;

	@PUT
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorSetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, JsonValue params) throws ProcessingException;

	@DELETE
	@Path("actor/{type}/{id}/state/{key}")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDeleteState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, @QueryParam("nilOnAbsent") boolean nilOnAbsent) throws ProcessingException;

	@GET
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response actorGetAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;

	@POST
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorSetMultipleState(@PathParam("type") String type, @PathParam("id") String id, JsonObject updates) throws ProcessingException;

	@DELETE
	@Path("actor/{type}/{id}/state")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	@Produces(MediaType.TEXT_PLAIN)
	public Response actorDeleteAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;

	/*
	 * Events
	 */

	@POST
	@Path("event/{topic}/publish")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response publish(@PathParam("topic") String topic) throws ProcessingException;

	@POST
	@Path("event/{topic}/subscribe")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response subscribe(@PathParam("topic") String topic) throws ProcessingException;

	@POST
	@Path("event/{topic}/unsubscribe")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response unsubscribe(@PathParam("topic") String topic) throws ProcessingException;

	/*
	 * System
	 */

	@POST
	@Path("system/shutdown")
	@Retry(maxRetries = KarConfig.MAX_RETRY)
	public Response shutdown() throws ProcessingException;
}
