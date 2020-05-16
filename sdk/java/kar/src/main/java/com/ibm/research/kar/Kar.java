package com.ibm.research.kar;

import java.net.URI;

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

		System.out.println("Port is " + port);

		if (port != null && !port.trim().isEmpty()) {
			baseURIStr = baseURIStr+":"+port+"/";
		} else {
			baseURIStr = baseURIStr+":"+DEFAULT_PORT+"/";
		}


		System.out.println("Sidecar location is " + baseURIStr);

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

	// asynchronous actor invocation, returns once invoke is scheduled
	public void actorTell(String type, String id, String path, JsonValue params) throws ProcessingException {
		karClient.actorTell(type, id, path, params);
	}

	// synchronous actor invocation with explicit session: returns result of the actor method
	public JsonValue actorCall(String type, String id,  String path, String session, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(type, id, path, session, params);
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null;
		}
  }

	// synchronous actor invocation: returns the result of the actor method
	public JsonValue actorCall(String type, String id,  String path, JsonValue params) throws ProcessingException {
		Response response = karClient.actorCall(type, id, path, null, params);
		if (response.hasEntity()) {
			return response.readEntity(JsonValue.class);
		} else {
			return null;
		}
	}

	/*
	 * Reminder Operations
	 */
	public Response actorCancelReminders(String type, String id) throws ProcessingException {
		return karClient.actorCancelReminders(type, id);

  }

  public Response actorCancelReminder(String type, String id, String reminderId) throws ProcessingException {
		return karClient.actorCancelReminder(type, id, reminderId, true);

  }

	public Response actorGetReminders(String type, String id) throws ProcessingException {
		return karClient.actorGetReminders(type, id);

  }

  public Response actorGetReminder(String type, String id, String reminderId) throws ProcessingException {
		return karClient.actorGetReminder(type, id, reminderId, true);

	}

	// FIXME:  Need to take targetTime and period as paramters and properly serialize
	public Response actorScheduleReminder(String type, String id, String path, String reminderId, JsonValue params) throws ProcessingException {
		JsonObjectBuilder builder = Json.createObjectBuilder();
		builder.add("path", "/"+path);
		builder.add("data", params);
    JsonObject requestBody =  builder.build();
		return karClient.actorScheduleReminder(type, id, requestBody);
	}

	/*
	 * Actor State Operations
	 */
	public Response actorGetState( String type,  String id,  String key) throws ProcessingException {
		return karClient.actorGetState(type, id, key, true);
	}

	public Response actorSetState(String type,  String id,  String key, JsonValue params) throws ProcessingException {
		return karClient.actorSetState(type, id, key, params);
	}

	public Response actorDeleteState(String type,  String id,  String key) throws ProcessingException {
		return karClient.actorDeleteState(type, id, key, true);
	}

	public Response actorGetAllState(String type,  String id) throws ProcessingException {
		return karClient.actorGetAllState(type, id);
	}

	public Response actorDeleteAllState(String type,  String id) throws ProcessingException {
		return karClient.actorDeleteAllState(type, id);
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
