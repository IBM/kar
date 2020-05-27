package com.ibm.research.kar;

import java.net.URI;
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

import org.eclipse.microprofile.rest.client.RestClientBuilder;

public class Kar {

	private static final Logger logger = Logger.getLogger(Kar.class.getName());

	private static KarRest karClient = buildRestClient();

	private Kar() {}

	/*
	 * Generate REST client (used when injection not possible, e.g. tests)
	 */
	private static KarRest buildRestClient() {

		String baseURIStr = "http://localhost";

		String port =  System.getenv("KAR_RUNTIME_PORT");
		logger.fine("KAR_RUNTIME_PORT set to " + port);


		if (port != null && !port.trim().isEmpty()) {
			baseURIStr = baseURIStr+":"+port+"/";
		} else {
			baseURIStr = baseURIStr+":"+KarConfig.DEFAULT_PORT+"/";
		}


		logger.fine("Sidecar location set to " + baseURIStr);

		URI baseURI = URI.create(baseURIStr);

		return  RestClientBuilder.newBuilder()
				.baseUri(baseURI)
				.readTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.build(KarRest.class);
	}

	private static JsonArray packArgs(JsonValue[] args) {
		JsonArrayBuilder ja = Json.createArrayBuilder();
		for (JsonValue a: args) {
			ja.add(a);
		}
		return ja.build();
	}

	private static JsonValue toValue(Response response) {
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null; // TODO: Should this be null or JSONValue.NULL?
		}
	}

	/******************
	 * Public methods
	 ******************/

	// asynchronous service invocation, returns once invoke is scheduled
	public static void tell(String service, String path, JsonValue params) throws ProcessingException {
		karClient.tell(service, path, params);
	}

	// synchronous service invocation, returns result of invoking the service
	public static JsonValue call(String service, String path, JsonValue params) throws ProcessingException {
		Response response = karClient.call(service, path, params);
		return toValue(response);
	}

	// Get a reference to an actor instance to use in subsequent actor operations.
	public static ActorRef actorRef(String type, String id) {
		return new ActorRefImpl(type, id);
	}

	// asynchronous actor invocation, returns once invoke is scheduled
	public static void actorTell(ActorRef p, String path, JsonValue... args) throws ProcessingException {
		karClient.actorTell(p.getType(), p.getId(), path, packArgs(args));
	}

	// synchronous actor invocation with explicit session: returns result of the actor method
	public static JsonValue actorCall(String callingSession, ActorRef p,  String path, JsonValue... args) throws ProcessingException {
		Response response = karClient.actorCall(p.getType(), p.getId(), path, callingSession, packArgs(args));
		return toValue(response);
	}

	// synchronous actor invocation: returns the result of the actor method
	public static JsonValue actorCall(ActorRef p, String path, JsonValue... args) throws ProcessingException {
		Response response = karClient.actorCall(p.getType(), p.getId(), path, null, packArgs(args));
		return toValue(response);
	}

	/*
	 * Reminder Operations
	 */
	public static Response actorCancelReminders(ActorRef p) throws ProcessingException {
		return karClient.actorCancelReminders(p.getType(), p.getId());

	}

	public static Response actorCancelReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorCancelReminder(p.getType(), p.getId(), reminderId, true);

	}

	public static Response actorGetReminders(ActorRef p) throws ProcessingException {
		return karClient.actorGetReminders(p.getType(), p.getId());

	}

	public static Response actorGetReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorGetReminder(p.getType(), p.getId(), reminderId, true);
	}

	// FIXME:  Need to take targetTime and period as paramters and properly serialize
	public static Response actorScheduleReminder(ActorRef p, String path, String reminderId, JsonValue params) throws ProcessingException {
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/"+path);
		builder.add("data", params);
		JsonObject requestBody =  builder.build();
		return karClient.actorScheduleReminder(p.getType(), p.getId(), reminderId, requestBody);
	}

	/*
	 * Actor State Operations
	 */
	public static JsonValue actorGetState(ActorRef p,  String key) {
		JsonValue value;
		try {
			Response resp = karClient.actorGetState(p.getType(), p.getId(), key, true);
			return toValue(resp);
		} catch (ProcessingException e) {
			value = JsonValue.NULL;
		}
		return value;
	}

	// TODO: return result of kar api call which is number of new entries created?
	public static void actorSetState(ActorRef p,  String key, JsonValue value) {
		karClient.actorSetState(p.getType(), p.getId(), key, value);
	}

	// TODO: return boolean based on 0/1 result of kar api call?
	public static void actorDeleteState(ActorRef p,  String key) {
		karClient.actorDeleteState(p.getType(), p.getId(), key, true);
	}

	public static Response actorGetAllState(ActorRef p) {
		return karClient.actorGetAllState(p.getType(), p.getId());
	}

	// TODO: return result of kar api call which is number of entries actually deleted?
	public static void actorDeleteAllState(ActorRef p) {
		karClient.actorDeleteAllState(p.getType(), p.getId());
	}

	// Events
	public static Response subscribe(String topic) throws ProcessingException {
		return karClient.subscribe(topic);
	}

	public static Response unsubscribe(String topic) throws ProcessingException {
		return karClient.unsubscribe(topic);
	}

	public static Response publish(String topic) throws ProcessingException {
		return karClient.publish(topic);
	}

	// System

	public static Response health() {
		return karClient.health();
	}

	public static Response kill() {
		return karClient.kill();
	}
}
