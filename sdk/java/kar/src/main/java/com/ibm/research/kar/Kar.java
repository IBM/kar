package com.ibm.research.kar;

import java.net.URI;

import javax.enterprise.context.ApplicationScoped;
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.ws.rs.PathParam;
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
	
	// asynchronous service invocation, returns "OK" immediately
	public Response tell(String service, String path, JsonObject params) throws ProcessingException {
		return karClient.tell(service, path, params);
	}

	// synchronous service invocation, returns invocation result
	public Response call(String service, String path, JsonObject params) throws ProcessingException {
		return karClient.call(service, path, params);
	}

	// asynchronous actor invocation, returns "OK" immediately
	public Response actorTell(String type, String id, String path, JsonObject params) throws ProcessingException {
		return karClient.actorTell(type, id, path, params);
	}

	// synchronous actor invocation with explicit session: returns invocation result
	public Response actorCall(String type, String id,  String path, String session, JsonObject params) throws ProcessingException {
		return karClient.actorCall(type, id, path, session, params);
  }

	// synchronous actor invocation: returns invocation result
	public Response actorCall(String type, String id,  String path, JsonObject params) throws ProcessingException {
		return karClient.actorCall(type, id, path, null, params);
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

	public Response actorScheduleReminder(String type, String id, String path, JsonObject params) throws ProcessingException {
 
		JsonObjectBuilder builder = Json.createObjectBuilder();
        
		builder.add("path", path);
        params.entrySet().
                forEach(e -> builder.add(e.getKey(), e.getValue()));
    
        JsonObject paramsWithPath =  builder.build();
		return karClient.actorScheduleReminder(type, id, paramsWithPath);
	}

	/*
	 * Actor State Operations
	 */
	public Response actorGetState( String type,  String id,  String key) throws ProcessingException {
		return karClient.actorGetState(type, id, key, true);
	}

	public Response actorSetState(String type,  String id,  String key, JsonObject params) throws ProcessingException {
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
