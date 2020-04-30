package com.ibm.research.kar;


import java.util.Map;

import javax.enterprise.inject.Default;
import javax.json.JsonObject;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.faulttolerance.Retry;
import org.eclipse.microprofile.rest.client.inject.RegisterRestClient;


@Default
@RegisterRestClient(configKey = "kar", baseUri = Kar.DEFAULT_URI)
@Path("kar/v1")
public interface KarRest extends AutoCloseable {
	
	int maxRetry = 10;

	/*
	 * Services
	 */
	
	// asynchronous service invocation, returns "OK" immediately
	@POST
	@Path("service/{service}/tell/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	@Retry(maxRetries = maxRetry)
	public Response tell(@PathParam("service") String service, @PathParam("path") String path, JsonObject params) throws ProcessingException;
	
	// synchronous service invocation, returns invocation result
	@POST
	@Path("service/{service}/call/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	@Retry(maxRetries = maxRetry)
	public Response call(@PathParam("service") String service, @PathParam("path") String path, JsonObject params) throws ProcessingException;

	
    /*
     * Actors
     */
	
	// asynchronous actor invocation, returns "OK" immediately
    @POST
	@Path("actor/{type}/{id}/tell/{path}")
    @Retry(maxRetries = maxRetry)
	public Response actorTell(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, JsonObject params) throws ProcessingException;

    // synchronous actor invocation: returns invocation result
    @POST
   	@Path("actor/{type}/{id}/call/{path}")
    @Retry(maxRetries = maxRetry)
   	public Response actorCall(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, JsonObject params) throws ProcessingException;

    // Request the migration of an actor
    @POST
   	@Path("actor/{type}/{id}/migrate")
    @Retry(maxRetries = maxRetry)
   	public Response actorMigrate(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;

    //
    // Actor Reminder operations
    //
 
    @DELETE
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorCancelReminder(@PathParam("type") String type, @PathParam("id") String id, JsonObject params) throws ProcessingException;
    
    @GET
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetReminder(@PathParam("type") String type, @PathParam("id") String id, JsonObject params) throws ProcessingException;
    
    @POST
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorScheduleReminder(@PathParam("type") String type, @PathParam("id") String id, JsonObject params) throws ProcessingException;


    //
    // Actor state operations
    //
    
    @GET
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;

    @PUT
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorSetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, JsonObject params) throws ProcessingException;
    
    @DELETE
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;
    
    @GET
   	@Path("actor/{type}/{id}/state")
    @Retry(maxRetries = maxRetry)
    public Response actorGetAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
    @POST
   	@Path("actor/{type}/{id}/state")
    @Retry(maxRetries = maxRetry)
    public Response actorSetAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
    @DELETE
   	@Path("actor/{type}/{id}/state")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
	/*
	 * Events
	 */
    
	@POST
	@Path("event/{topic}/publish")
	@Retry(maxRetries = maxRetry)
	public Response publish(@PathParam("topic") String topic) throws ProcessingException;
	
	@POST
	@Path("event/{topic}/subscribe")
	@Retry(maxRetries = maxRetry)
	public Response subscribe(@PathParam("topic") String topic) throws ProcessingException;
	
	@POST
	@Path("event/{topic}/unsubscribe")
	@Retry(maxRetries = maxRetry)
	public Response unsubscribe(@PathParam("topic") String topic) throws ProcessingException;
	
	/*
	 * System 
	 */
	
	@POST
	@Path("system/broadcast/{path}")
	@Retry(maxRetries = maxRetry)
	public Response broadcast(@PathParam("path") String path, JsonObject params) throws ProcessingException;
   
	@GET
	@Path("system/health")
	@Retry(maxRetries = maxRetry)
	public Response health() throws ProcessingException;
	
	@POST
	@Path("system/kill")
	@Retry(maxRetries = maxRetry)
	public Response kill() throws ProcessingException;
	
	@POST
	@Path("system/killAll")
	@Retry(maxRetries = maxRetry)
	public Response killAll() throws ProcessingException;
	

  	
}