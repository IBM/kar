package com.ibm.research.kar;

import java.net.URI;
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

	public static final String DEFAULT_PORT = "3500";

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
			baseURIStr = baseURIStr+":"+DEFAULT_PORT+"/";
		}


		logger.fine("Sidecar location set to " + baseURIStr);

		URI baseURI = URI.create(baseURIStr);

		return  RestClientBuilder.newBuilder()
				.baseUri(baseURI)
				.build(KarRest.class);
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
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null;
		}
	}

	// Get a reference to an actor instance to use in subsequent actor operations.
	public ActorRef actorRef(String type, String id) {
		return new ActorRef(type, id);
	}

	// asynchronous actor invocation, returns once invoke is scheduled
	public void actorTell(ActorRef p, String path, JsonValue params) throws ProcessingException {
		karClient.actorTell(p.type, p.id, path, params);
	}

	// synchronous actor invocation with explicit session: returns result of the actor method
	public JsonValue actorCall(String callingSession, ActorRef p,  String path, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(p.type, p.id, path, callingSession, params);
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null;
		}
	}

	// synchronous actor invocation: returns the result of the actor method
	public JsonValue actorCall(ActorRef p, String path, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(p.type, p.id, path, null, params);
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null;
		}
	}

	/*
	 * Reminder Operations
	 */
	public Response actorCancelReminders(ActorRef p) throws ProcessingException {
		return karClient.actorCancelReminders(p.type, p.id);

	}

	public Response actorCancelReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorCancelReminder(p.type, p.id, reminderId, true);

	}

	public Response actorGetReminders(ActorRef p) throws ProcessingException {
		return karClient.actorGetReminders(p.type, p.id);

	}

	public Response actorGetReminder(ActorRef p, String reminderId) throws ProcessingException {
		return karClient.actorGetReminder(p.type, p.id, reminderId, true);
	}

	// FIXME:  Need to take targetTime and period as paramters and properly serialize
	public Response actorScheduleReminder(ActorRef p, String path, String reminderId, JsonValue params) throws ProcessingException {
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/"+path);
		builder.add("data", params);
		JsonObject requestBody =  builder.build();
		return karClient.actorScheduleReminder(p.type, p.id, requestBody);
	}

	/*
	 * Actor State Operations
	 */
	public Response actorGetState(ActorRef p,  String key) throws ProcessingException {
		return karClient.actorGetState(p.type, p.id, key, true);
	}

	public Response actorSetState(ActorRef p,  String key, JsonValue params) throws ProcessingException {
		return karClient.actorSetState(p.type, p.id, key, params);
	}

	public Response actorDeleteState(ActorRef p,  String key) throws ProcessingException {
		return karClient.actorDeleteState(p.type, p.id, key, true);
	}

	public Response actorGetAllState(ActorRef p) throws ProcessingException {
		return karClient.actorGetAllState(p.type, p.id);
	}

	public Response actorDeleteAllState(ActorRef p) throws ProcessingException {
		return karClient.actorDeleteAllState(p.type, p.id);
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
