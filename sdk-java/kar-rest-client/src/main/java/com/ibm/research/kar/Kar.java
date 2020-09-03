package com.ibm.research.kar;

import java.net.URI;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.Reminder;
import com.ibm.research.kar.actor.exceptions.ActorExceptionMapper;
import com.ibm.research.kar.actor.exceptions.ActorMethodInvocationException;
import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;

import org.eclipse.microprofile.rest.client.RestClientBuilder;
import org.glassfish.json.jaxrs.JsonValueBodyReader;
import org.glassfish.json.jaxrs.JsonValueBodyWriter;

public class Kar {

	private static final Logger logger = Logger.getLogger(Kar.class.getName());

	private static KarRest karClient = buildRestClient();

	private Kar() { }

	/*
	 * Generate REST client (used when injection not possible, e.g. tests)
	 */
	private static KarRest buildRestClient() {
		String baseURIStr = "http://localhost";

		String port = System.getenv("KAR_RUNTIME_PORT");
		logger.fine("KAR_RUNTIME_PORT set to " + port);

		if (port != null && !port.trim().isEmpty()) {
			baseURIStr = baseURIStr + ":" + port + "/";
		} else {
			baseURIStr = baseURIStr + ":" + KarConfig.DEFAULT_PORT + "/";
		}

		logger.fine("Sidecar location set to " + baseURIStr);

		URI baseURI = URI.create(baseURIStr);

		RestClientBuilder builder = RestClientBuilder.newBuilder().baseUri(baseURI);

		// If running in standalone mode, add JsonValue serializers by hand
		if (!isRunningEmbedded()) {
			builder
			.register(JsonValueBodyReader.class)
			.register(JsonValueBodyWriter.class);
		}

		return builder
				.register(ActorExceptionMapper.class)
				.readTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS).build(KarRest.class);
	}

	private static boolean isRunningEmbedded() {
		return (System.getProperty("wlp.server.name") != null);
	}

	private static JsonArray packArgs(JsonValue[] args) {
		JsonArrayBuilder ja = Json.createArrayBuilder();
		for (JsonValue a : args) {
			ja.add(a);
		}
		return ja.build();
	}

	private static JsonValue actorUnwrap(JsonValue v) throws ActorMethodInvocationException {
		JsonObject o = v.asJsonObject();
		if (o.containsKey("error")) {
			String message = o.containsKey("message") ? o.getString("message") : "Unknown error";
			String stack = o.containsKey("stack") ? o.getString("stack") : "";
			throw new ActorMethodInvocationException(message + ": "+stack);
		} else {
			return o.containsKey("value") ? o.get("value") : JsonValue.NULL;
		}
	}

	private static Object toValue(Response response) {
		if (response.hasEntity()) {
			MediaType type = response.getMediaType();
			if (type.equals(MediaType.APPLICATION_JSON_TYPE)) {
				return response.readEntity(JsonValue.class);
			} else if (type.equals(MediaType.TEXT_PLAIN_TYPE)) {
				return response.readEntity(String.class);
			} else {
				return JsonValue.NULL;
			}
		} else {
			return JsonValue.NULL;
		}
	}

	private static int toInt(Response response) {
		if (response.getStatus() == Status.OK.getStatusCode() && response.hasEntity()) {
			return response.readEntity(java.lang.Integer.TYPE);
		} else {
			return 0;
		}
	}

	private static Reminder[] toReminderArray(Response response) {
		try {
			ArrayList<Reminder> res = new ArrayList<Reminder>();
			JsonArray ja = ((JsonValue)toValue(response)).asJsonArray();
			for (JsonValue jv : ja) {
				try {
					JsonObject jo = jv.asJsonObject();
					String actorType = jo.getJsonObject("Actor").getString("Type");
					String actorId = jo.getJsonObject("Actor").getString("ID");
					String id = jo.getString("id");
					String path = jo.getString("path");
					String targetTimeString = jo.getString("targetTime");
					Instant targetTime = Instant.parse(targetTimeString);
					Duration period = null;
					if (jo.get("period") != null) {
						long nanos = ((JsonNumber)jo.get("period")).longValueExact();
						period = Duration.ofNanos(nanos);
					}
					String encodedData = jo.getString("encodedData");
					Reminder r = new Reminder(actorRef(actorType, actorId), id, path, targetTime, period, encodedData);
					res.add(r);
				} catch (ClassCastException e) {
					logger.warning("toReminderArray: Dropping unexpected element "+jv);
				}
			}
			return res.toArray(new Reminder[res.size()]);
		} catch (ClassCastException e) {
			return new Reminder[0];
		}
	}

	private static final class ActorRefImpl implements ActorRef {
		final String type;
		final String id;

		ActorRefImpl(String type, String id) {
			this.type = type;
			this.id = id;
		}

		@Override
		public String getType() {
			return type;
		}

		@Override
		public String getId() {
			return id;
		}
	}


	/******************
	 * Public methods
	 ******************/

	/*
	 * Lower-level REST operations on a KAR Service
	 */

	/**
	 * Synchronous REST DELETE
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static Response restDelete(String service, String path) throws ProcessingException {
		return karClient.callDelete(service, path);
	}

	/**
	 * Asynchronous REST DELETE
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restDeleteAsync(String service, String path) throws ProcessingException {
		return karClient.callAsyncDelete(service, path);
	}

	/**
	 * Synchronous REST GET
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static Response restGet(String service, String path) throws ProcessingException {
		return karClient.callGet(service, path);
	}

	/**
	 * Asynchronous REST GET
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restGetAsync(String service, String path) throws ProcessingException {
		return karClient.callAsyncGet(service, path);
	}

	/**
	 * Synchronous REST HEAD
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static Response restHead(String service, String path) throws ProcessingException {
		return karClient.callHead(service, path);
	}

	/**
	 * Asynchronous REST HEAD
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restHeadAsync(String service, String path) throws ProcessingException {
		return karClient.callAsyncHead(service, path);
	}

	/**
	 * Synchronous REST OPTIONS
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static Response restOptions(String service, String path) throws ProcessingException {
		return karClient.callOptions(service, path, JsonValue.NULL);
	}

	/**
	 * Synchronous REST OPTIONS
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static Response restOptions(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callOptions(service, path, body);
	}

	/**
	 * Asynchronous REST OPTIONS
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restOptionsAsync(String service, String path) throws ProcessingException {
		return karClient.callAsyncOptions(service, path, JsonValue.NULL);
	}

	/**
	 * Asynchronous REST OPTIONS
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restOptionsAsync(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callAsyncOptions(service, path, body);
	}

	/**
	 * Synchronous REST PATCH
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static Response restPatch(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callPatch(service, path, body);
	}

	/**
	 * Asynchronous REST PATCH
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restPatchAsync(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callAsyncPatch(service, path, body);
	}

	/**
	 * Synchronous REST POST
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static Response restPost(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callPost(service, path, body);
	}

	/**
	 * Asynchronous REST POST
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restPostAsync(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callAsyncPost(service, path, body);
	}

	/**
	 * Synchronous REST PUT
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static Response restPut(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callPut(service, path, body);
	}

	/**
	 * Asynchronous REST PUT
	 *
	 * @param service The name of the service.
	 * @param path    The service endpoint.
	 * @param body    The request body.
	 * @return The response returned by the target service.
	 */
	public static CompletionStage<Response> restPutAsync(String service, String path, JsonValue body) throws ProcessingException {
		return karClient.callAsyncPut(service, path, body);
	}

	/*
	 * Higher-level Service call/tell operations that hide the REST layer
	 */

	/**
	 * Asynchronous service invocation; returns as soon as the invocation has been initiated.
	 *
	 * @param service The name of the service to invoke.
	 * @param path    The service endpoint to invoke.
	 * @param body    The request body with which to invoke the service endpoint.
	 */
	public static void tell(String service, String path, JsonValue body) {
		karClient.tellPost(service, path, body);
	}

	/**
	 * Synchronous service invocation
	 *
	 * @param service The name of the service to invoke.
	 * @param path    The service endpoint to invoke.
	 * @param body    The request body with which to invoke the service endpoint.
	 * @return The result returned by the target service.
	 */
	public static Object call(String service, String path, JsonValue body) {
		Response resp = karClient.callPost(service, path, body);
		return toValue(resp);
	}

	/**
	 * Aynchronous service invocation with eventual access to the result of the invocation
	 *
	 * @param service The name of the service to invoke.
	 * @param path    The service endpoint to invoke.
	 * @param body    The request body with which to invoke the service endpoint.
	 * @return A CompletionStage containing the result of invoking the target service.
	 */
	public static CompletionStage<Object> callAsync(String service, String path, JsonValue body) {
		return karClient.callAsyncPut(service, path, body).thenApply(response -> toValue(response));
	}

	/**
	 * Construct an ActorRef that represents a specific Actor instance.
	 *
	 * @param type The type of the Actor instance
	 * @param id   The instance id of the Actor instance
	 * @return An ActorRef representing the Actor instance.
	 */
	public static ActorRef actorRef(String type, String id) {
		return new ActorRefImpl(type, id);
	}

	/**
	 * Asynchronous actor invocation; returns as soon as the invocation has been initiated.
	 *
	 * @param actor The target actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 */
	public static void actorTell(ActorRef actor, String path, JsonValue... args) {
		karClient.actorTell(actor.getType(), actor.getId(), path, packArgs(args));
	}

	/**
	 * Synchronous actor invocation where the invoked method will execute as part of the current session.
	 *
	 * @param caller The calling actor.
	 * @param actor The target actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return The result of the invoked actor method.
	 */
	public static JsonValue actorCall(ActorInstance caller, ActorRef actor, String path, JsonValue... args) throws ActorMethodNotFoundException, ActorMethodInvocationException {
		return actorUnwrap(karClient.actorCall(actor.getType(), actor.getId(), path, caller.getSession(), packArgs(args)));

	}

	/**
	 * Synchronous actor invocation where the invoked method will execute as part of the specified session.
	 *
	 * @param session The session in which to execute the actor method
	 * @param actor The target actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return The result of the invoked actor method.
	 */
	public static JsonValue actorCall(String session, ActorRef actor, String path, JsonValue... args) throws ActorMethodNotFoundException, ActorMethodInvocationException{
		return actorUnwrap(karClient.actorCall(actor.getType(), actor.getId(), path, session, packArgs(args)));
	}

	/**
	 * Synchronous actor invocation where the invoked method will execute in a new session.
	 *
	 * @param actor The target Actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return The result of the invoked actor method.
	 */
	public static JsonValue actorCall(ActorRef actor, String path, JsonValue... args) throws ActorMethodNotFoundException, ActorMethodInvocationException {
		return actorUnwrap(karClient.actorCall(actor.getType(), actor.getId(), path, null, packArgs(args)));
	}

	/**
	 * Asynchronous actor invocation with eventual access to the result of the invocation.
	 *
	 * @param actor The target Actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return A CompletionStage containing the result of invoking the target service.
	 */
	public static CompletionStage<JsonValue> actorCallAsync(ActorRef actor, String path, JsonValue... args) throws ActorMethodNotFoundException {
		return karClient.actorCallAsync(actor.getType(), actor.getId(), path, null, packArgs(args));
	}

	/*
	 * Reminder Operations
	 */

	/**
	 * Cancel all reminders for an Actor instance.
	 *
	 * @param actor The Actor instance.
	 * @return The number of reminders that were cancelled.
	 */
	public static int actorCancelAllReminders(ActorRef actor) {
		Response response = karClient.actorCancelReminders(actor.getType(), actor.getId());
		return toInt(response);
	}

	/**
	 * Cancel a specific reminder for an Actor instance.
	 *
	 * @param actor      The Actor instance.
	 * @param reminderId The id of a specific reminder to cancel
	 * @return The number of reminders that were cancelled.
	 */
	public static int actorCancelReminder(ActorRef actor, String reminderId) {
		Response response = karClient.actorCancelReminder(actor.getType(), actor.getId(), reminderId, true);
		return toInt(response);
	}

	/**
	 * Get all reminders for an Actor instance.
	 *
	 * @param actor      The Actor instance.
	 * @return An array of matching reminders
	 */
	public static Reminder[] actorGetAllReminders(ActorRef actor) {
		Response response = karClient.actorGetReminders(actor.getType(), actor.getId());
		return toReminderArray(response);
	}

	/**
	 * Get a specific reminder for an Actor instance.
	 *
	 * @param actor      The Actor instance.
	 * @param reminderId The id of a specific reminder to cancel
	 * @returns An array of matching reminders
	 */
	public static Reminder[] actorGetReminder(ActorRef actor, String reminderId) {
		Response response = karClient.actorGetReminder(actor.getType(), actor.getId(), reminderId, true);
		return toReminderArray(response);
	}

	/**
	 * Schedule a reminder for an Actor instance.
	 *
	 * @param actor      The Actor instance.
	 * @param path       The actor method to invoke when the reminder fires.
	 * @param reminderId The id of the reminder being scheduled
	 * @param targetTime The earliest time at which the reminder should be delivered
	 * @param period     For periodic reminders, a String that is compatible with GoLang's Duration
	 * @param args       The arguments with which to invoke the actor method.
	 */
	public static void actorScheduleReminder(ActorRef actor, String path, String reminderId, Instant targetTime, Duration period, JsonValue... args) {
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/" + path);
		builder.add("targetTime", targetTime.toString());

		if (period != null) {
			// Sigh.  Encode in a way that GoLang will understand since it sadly doesn't actually implement ISO-8601
			String goPeriod = "";
			if (period.toHours() > 0) {
				goPeriod += period.toHours()+"h";
				period.minusHours(period.toHours());
			}
			if (period.toMinutes() > 0) {
				goPeriod += period.toMinutes()+"m";
				period.minusMinutes(period.toMinutes());
			}
			goPeriod += period.getSeconds()+"s";
			builder.add("period", goPeriod);
		}
		builder.add("data", packArgs(args));
		JsonObject requestBody = builder.build();

		karClient.actorScheduleReminder(actor.getType(), actor.getId(), reminderId, requestBody);
	}

	/*
	 * Actor State Operations
	 */

	/**
	 * Get one value from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to use to access the instance's state
	 * @return The value associated with `key`
	 */
	public static JsonValue actorGetState(ActorRef actor, String key) {
		JsonValue value;
		try {
			Response resp = karClient.actorGetState(actor.getType(), actor.getId(), key, true);
			return (JsonValue)toValue(resp);
		} catch (ProcessingException e) {
			value = JsonValue.NULL;
		}
		return value;
	}

	/**
	 * Get one value from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to use to access the instance's state
	 * @param subkey The subkey to use to access the instance's state
	 * @return The value associated with `key/subkey`
	 */
	public static JsonValue actorGetState(ActorRef actor, String key, String subkey) {
		JsonValue value;
		try {
			Response resp = karClient.actorGetWithSubkeyState(actor.getType(), actor.getId(), key, subkey, true);
			return (JsonValue)toValue(resp);
		} catch (ProcessingException e) {
			value = JsonValue.NULL;
		}
		return value;
	}

	/**
	 * Check to see if an entry exists in an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to check against the instance's state
	 * @return `true` if the actor instance has a value defined for `key`, `false` otherwise.
	 */
	public static boolean actorContainsState(ActorRef actor, String key) {
		try {
			Response resp = karClient.actorHeadState(actor.getType(), actor.getId(), key);
			return resp.getStatus() == Status.OK.getStatusCode();
		} catch (ActorMethodNotFoundException e) { // FIXME: This is an artifact of overly broad exceptiob remapping.
			return false;
		}
	}

	/**
	 * Check to see if an entry exists in an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to check against the instance's state
	 * @param subkey The subkey to check against the instance's state
	 * @return `true` if the actor instance has a value defined for `key/subkey`, `false` otherwise.
	 */
	public static boolean actorContainsState(ActorRef actor, String key, String subkey) {
		try {
			Response resp = karClient.actorHeadWithSubkeyState(actor.getType(), actor.getId(), key, subkey);
			return resp.getStatus() == Status.OK.getStatusCode();
		} catch (ActorMethodNotFoundException e) { // FIXME: This is an artifact of overly broad exceptiob remapping.
			return false;
		}
	}

	/**
	 * Store one value to an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to use to access the instance's state
	 * @param value The value to store
	 * @return The number of new state entries created by this store (0 or 1)
	 */
	public static int actorSetState(ActorRef actor, String key, JsonValue value) {
		Response response = karClient.actorSetState(actor.getType(), actor.getId(), key, value);
		return toInt(response);
	}

	/**
	 * Store one value to an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to use to access the instance's state
	 * @param subkey The subkey to use to access the instance's state
	 * @param value The value to store at `key/subkey`
	 * @return The number of new state entries created by this store (0 or 1)
	 */
	public static int actorSetState(ActorRef actor, String key, String subkey, JsonValue value) {
		Response response = karClient.actorSetWithSubkeyState(actor.getType(), actor.getId(), key, subkey, value);
		return toInt(response);
	}

	/**
	 * Store multiple values to an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param updates A map containing the state updates to perform
	 * @return The number of new state entries created by this operation
	 */
	public static int actorSetMultipleState(ActorRef actor, Map<String,JsonValue> updates) {
		JsonObjectBuilder jb = Json.createObjectBuilder();
		for (Entry<String,JsonValue> e : updates.entrySet()) {
			jb.add(e.getKey(), e.getValue());
		}
		JsonObject jup = jb.build();
		Response response = karClient.actorSetMultipleState(actor.getType(), actor.getId(), jup);
		return toInt(response);
	}

	/**
	 * Remove one value from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to delete
	 * @return  `1` if an entry was actually removed and `0` if there was no entry for `key`.
	 */
	public static int actorDeleteState(ActorRef actor, String key) {
		Response response = karClient.actorDeleteState(actor.getType(), actor.getId(), key, true);
		return toInt(response);
	}

	/**
	 * Remove one value from an Actor's state
	 *
	 * @param actor  The Actor instance.
	 * @param key    The key of the entry to delete
	 * @param subkey The subkey of the entry to delete
	 * @return  `1` if an entry was actually removed and `0` if there was no entry for `key`.
	 */
	public static int actorDeleteState(ActorRef actor, String key, String subkey) {
		Response response = karClient.actorDeleteWithSubkeyState(actor.getType(), actor.getId(), key, subkey, true);
		return toInt(response);
	}

	/**
	 * Get all the key-value mappings from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @return A map representing the Actor's state
	 */
	public static Map<String,JsonValue> actorGetAllState(ActorRef actor) {
		Response response = karClient.actorGetAllState(actor.getType(), actor.getId());
		try {
			return ((JsonValue)toValue(response)).asJsonObject();
		} catch (ClassCastException e) {
			return Collections.emptyMap();
		}
	}

	/**
	 * Remove all key-value pairs from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @return The number of removed key/value pairs
	 */
	public static int actorDeleteAllState(ActorRef actor) {
		Response response = karClient.actorDeleteAllState(actor.getType(), actor.getId());
		return toInt(response);
	}

/* WIP -- pieces of subkey API that still need to be added.  FIXME!

	public static int actorMapSize(ActorRef actor, String mapName) {
		return 0;
	}

	public static void actorMapClear(ActorRef actor, String mapName) {
	}

	public static Map<String, JsonValue> actorMapGetMap(ActorRef actor, String mapName) {
		return null;
	}

	public static void actorMapPutAll(ActorRef actor, String mapName, Map<String,JsonValue> map) {
	}

*/

	/*
	 * Events
	 */

	/**
	 * Subscribe a Service endpoint to a topic.
	 *
	 * @param topic The topic to which to subscribe
	 * @param path  The endpoint to invoke for each event received on the topic
	 * @param opts  TODO: Document expected structure
	 */
	public static Response subscribe(String topic) throws ProcessingException {
		return karClient.subscribe(topic); // FIXME: actually implement
	}

	/**
	 * Subscribe an Actor instance method to a topic.
	 *
	 * @param actor The Actor instance to subscribe
	 * @param topic The topic to which to subscribe
	 * @param path  The endpoint to invoke for each event received on the topic
	 * @param opts  TODO: Document expected structure
	 */
	public static Response actorSubscribe(String topic) throws ProcessingException {
		return karClient.subscribe(topic); // FIXME: actually implement
	}

	/**
	 * Unsubscribe from a topic.
	 *
	 * @param topic The topic to which to subscribe
	 * @param opts  TODO: Document expected structure
	 */
	public static Response unsubscribe(String topic) throws ProcessingException {
		return karClient.unsubscribe(topic); // FIXME: actually implement
	}

	/**
	 * Publish a CloudEvent to a topic
	 *
	 * TODO: Document this API when it stabalizes
	 */
	public static Response publish(String topic) throws ProcessingException {
		return karClient.publish(topic); // FIXME: actually implement
	}

	/*
	 * System
	 */

	/**
	 * Shutdown this sidecar.  Does not return.
	 */
	public static void shutdown() {
		karClient.shutdown();
	}
}