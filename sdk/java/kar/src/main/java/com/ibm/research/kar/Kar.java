package com.ibm.research.kar;

import java.net.URI;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import javax.enterprise.context.ApplicationScoped;
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.rest.client.RestClientBuilder;

@ApplicationScoped // change as needed
public class Kar {

	private final Logger logger = Logger.getLogger(Kar.class.getName());

	private KarRest karClient;

	public Kar() {
		karClient = buildRestClient();
	}

	/*
	 * Generate REST client (used when injection not possible, e.g. tests)
	 */
	public KarRest buildRestClient() {

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
	public void tell(String service, String path, JsonValue params) throws ProcessingException {
		karClient.tell(service, path, params);
	}

	// synchronous service invocation, returns result of invoking the service
	public JsonValue call(String service, String path, JsonValue params) throws ProcessingException {
		Response response = karClient.call(service, path, params);
		return toValue(response);
	}

	// Get a reference to an actor instance to use in subsequent actor operations.
	public ActorRef actorRef(String type, String id) {
		return new ActorRefImpl(type, id);
	}

	// asynchronous actor invocation, returns once invoke is scheduled
	public void actorTell(ActorRef p, String path, JsonValue params) throws ProcessingException {
		karClient.actorTell(p.getType(), p.getId(), path, params);
	}

	// synchronous actor invocation with explicit session: returns result of the actor method
	public JsonValue actorCall(String callingSession, ActorRef p,  String path, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(p.getType(), p.getId(), path, callingSession, params);
		return toValue(response);
	}

	// synchronous actor invocation: returns the result of the actor method
	public JsonValue actorCall(ActorRef p, String path, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(p.getType(), p.getId(), path, null, params);
		return toValue(response);
	}

	/*
	 * Reminder Operations
	 */
	public Response actorCancelReminders(ActorRef p) throws ProcessingException {
		return karClient.actorCancelReminders(p.getType(), p.getId());

	}

	public Response actorCancelReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorCancelReminder(p.getType(), p.getId(), reminderId, true);

	}

	public Response actorGetReminders(ActorRef p) throws ProcessingException {
		return karClient.actorGetReminders(p.getType(), p.getId());

	}

	public Response actorGetReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorGetReminder(p.getType(), p.getId(), reminderId, true);
	}

	// FIXME:  Need to take targetTime and period as paramters and properly serialize
	public Response actorScheduleReminder(ActorRef p, String path, String reminderId, JsonValue params) throws ProcessingException {
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/"+path);
		builder.add("data", params);
		JsonObject requestBody =  builder.build();
		return karClient.actorScheduleReminder(p.getType(), p.getId(), requestBody);
	}

	/*
	 * Actor State Operations
	 */
	public JsonValue actorGetState(ActorRef p,  String key) {
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
	public void actorSetState(ActorRef p,  String key, JsonValue value) {
		karClient.actorSetState(p.getType(), p.getId(), key, value);
	}

	// TODO: return boolean based on 0/1 result of kar api call?
	public void actorDeleteState(ActorRef p,  String key) {
		karClient.actorDeleteState(p.getType(), p.getId(), key, true);
	}

	public Response actorGetAllState(ActorRef p) {
		return karClient.actorGetAllState(p.getType(), p.getId());
	}

	// TODO: return result of kar api call which is number of entries actually deleted?
	public void actorDeleteAllState(ActorRef p) {
		karClient.actorDeleteAllState(p.getType(), p.getId());
	}

	// Events
	public Response subscribe(String topic) throws ProcessingException {
		return karClient.subscribe(topic);
	}

	public Response unsubscribe(String topic) throws ProcessingException {
		return karClient.unsubscribe(topic);
	}

	public Response publish(String topic) throws ProcessingException {
		return karClient.publish(topic);
	}

	// System

	public Response health() {
		return karClient.health();
	}

	public Response kill() {
		return karClient.kill();
	}
}
