package com.ibm.research.kar;

import java.net.URI;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.ws.rs.PathParam;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.rest.client.RestClientBuilder;
import org.eclipse.microprofile.rest.client.inject.RestClient;

@ApplicationScoped // change as needed
public class Kar {

	@Inject
	@RestClient
	private KarRest karClient;
	
	public static final String DEFAULT_URI = "http://localhost:3500/";
	
	/*
	 * Generate REST client (used when injection not possible, e.g. tests)
	 */
	public void buildRestClient() {
		URI baseURI = URI.create(Kar.DEFAULT_URI);
		
		karClient = RestClientBuilder.newBuilder()
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

	// synchronous actor invocation: returns invocation result
	public Response actorCall(String type, String id,  String path, JsonObject params) throws ProcessingException {
		return karClient.actorCall(type, id, path, params);
	}
	
	// migrate actor
	public Response actorMigrate(String type, String id) throws ProcessingException {
		return karClient.actorMigrate(type, id);
	}

	
	/*
	 * Reminder Operations
	 */
	public Response actorCancelReminder(String type, String id, JsonObject params) throws ProcessingException {
		return karClient.actorCancelReminder(type, id, params);

	}

	public Response actorGetReminder(String type, String id, JsonObject params) throws ProcessingException {
		return karClient.actorGetReminder(type, id, params);

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
    	return karClient.actorGetState(type, id, key);
    }

    public Response actorSetState(String type,  String id,  String key, JsonObject params) throws ProcessingException {
    	return karClient.actorSetState(type, id, key, params);
    }

    public Response actorDeleteState(String type,  String id,  String key) throws ProcessingException {
    	return karClient.actorDeleteState(type, id, key);
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
	// broadcast to all sidecars except for ours
	public Response broadcast(@PathParam("path") String path, Map<String,Object> params) throws ProcessingException {
		return karClient.broadcast(path, params);
	}
	
	public Response health() {
		return karClient.health();
	}
	
	public Response kill() {
		return karClient.kill();
	}
	
	public Response killAll() {
		return karClient.killAll();
	}
	

}
