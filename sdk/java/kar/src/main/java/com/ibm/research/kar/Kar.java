package com.ibm.research.kar;

import java.net.URI;
import java.util.Collections;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.actor.ActorRef;

import org.eclipse.microprofile.rest.client.RestClientBuilder;

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

		return RestClientBuilder.newBuilder().baseUri(baseURI)
				.readTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS).build(KarRest.class);
	}

	private static JsonArray packArgs(JsonValue[] args) {
		JsonArrayBuilder ja = Json.createArrayBuilder();
		for (JsonValue a : args) {
			ja.add(a);
		}
		return ja.build();
	}

	private static JsonValue toValue(Response response) {
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
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

	/**
	 * Asynchronous service invocation
	 *
	 * @param service The name of the service to invoke.
	 * @param path    The service endpoint to invoke.
	 * @param body    The request body with which to invoke the service endpoint.
	 */
	public static void tell(String service, String path, JsonValue body) {
		karClient.tell(service, path, body);
	}

	/**
	 * Synchronous service invocation
	 *
	 * @param service The name of the service to invoke.
	 * @param path    The service endpoint to invoke.
	 * @param body    The request body with which to invoke the service endpoint.
	 * @return The result returned by the target service.
	 */
	public static JsonValue call(String service, String path, JsonValue body) {
		Response response = karClient.call(service, path, body);
		return toValue(response);
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
	 * Asynchronous actor invocation
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
	 * @param callingSession The current session
	 * @param actor The target actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return The result of the invoked actor method.
	 */
	public static JsonValue actorCall(String callingSession, ActorRef actor, String path, JsonValue... args) {
		Response response = karClient.actorCall(actor.getType(), actor.getId(), path, callingSession, packArgs(args));
		return toValue(response);
	}

	/**
	 * Synchronous actor invocation where the invoked method will execute in a new session.
	 *
	 * @param actor The target Actor.
	 * @param path  The actor method to invoke.
	 * @param args  The arguments with which to invoke the actor method.
	 * @return The result of the invoked actor method.
	 */
	public static JsonValue actorCall(ActorRef actor, String path, JsonValue... args) {
		Response response = karClient.actorCall(actor.getType(), actor.getId(), path, null, packArgs(args));
		return toValue(response);
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
	 * @return An array of matching reminders FIXME: actually implement
	 */
	public static Response actorGetAllReminders(ActorRef actor) {
		return karClient.actorGetReminders(actor.getType(), actor.getId());
	}

	/**
	 * Get a specific reminder for an Actor instance.
	 *
	 * @param actor      The Actor instance.
	 * @param reminderId The id of a specific reminder to cancel
	 * @returns An array of matching reminders  FIXME: actually implement
	 */
	public static Response actorGetReminder(ActorRef actor, String reminderId) {
		return karClient.actorGetReminder(actor.getType(), actor.getId(), reminderId, true);
	}

	/**
	 * Schedule a reminder for an Actor instance.   FIXME: actually implement
	 *
	 * @param actor              The Actor instance.
	 * @param path               The actor method to invoke when the reminder fires.
	 * @param options.id         The id of the reminder being scheduled
	 * @param options.targetTime The earliest time at which the reminder should be
	 *                           delivered
	 * @param options.period     For periodic reminders, a string encoding a
	 *                           Duration representing the desired gap between
	 *                           successive reminders
	 * @param args               The arguments with which to invoke the actor
	 *                           method.
	 */
	public static Response actorScheduleReminder(ActorRef actor, String path, String reminderId, JsonValue params) {
		// FIXME: Need to take targetTime and period as paramters and properly serialize
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/" + path);
		builder.add("data", params);
		JsonObject requestBody = builder.build();
		return karClient.actorScheduleReminder(actor.getType(), actor.getId(), reminderId, requestBody);
	}

	/*
	 * Actor State Operations
	 */

	/**
	 * Get one value from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to get from the instance's state
	 * @return The value associated with `key`
	 */
	public static JsonValue actorGetState(ActorRef actor, String key) {
		JsonValue value;
		try {
			Response resp = karClient.actorGetState(actor.getType(), actor.getId(), key, true);
			return toValue(resp);
		} catch (ProcessingException e) {
			value = JsonValue.NULL;
		}
		return value;
	}

	/**
	 * Store one value to an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @param key   The key to get from the instance's state
	 * @param value The value to store
	 * @return The number of new state entries created by this store (0 or 1)
	 */
	public static int actorSetState(ActorRef actor, String key, JsonValue value) {
		Response response = karClient.actorSetState(actor.getType(), actor.getId(), key, value);
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
	 * Get all the key value pairs from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @return A map representing the Actor's state
	 */
	public static Map<String,JsonValue> actorGetAllState(ActorRef actor) {
		Response response = karClient.actorGetAllState(actor.getType(), actor.getId());
		try {
			return toValue(response).asJsonObject();
		} catch (ClassCastException e) {
			return Collections.emptyMap();
		}
	}

	/**
	 * Remove all key value pairs from an Actor's state
	 *
	 * @param actor The Actor instance.
	 * @return The number of removed key/value pairs
	 */
	public static int actorDeleteAllState(ActorRef actor) {
		Response response = karClient.actorDeleteAllState(actor.getType(), actor.getId());
		return toInt(response);
	}

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
