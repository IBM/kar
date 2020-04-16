package com.ibm.research.kar;


import java.util.Map;

import javax.enterprise.inject.Default;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.faulttolerance.Retry;
import org.eclipse.microprofile.rest.client.inject.RegisterRestClient;


@Default
@RegisterRestClient(configKey = "kar", baseUri = "http://localhost:3500/")
@Path("kar/v1")
public interface KarRest extends AutoCloseable {
	
	int maxRetry = 10;
	
	/*
	 * Public methods
	 */
	
	// asynchronous service invocation, returns "OK" immediately
	@POST
	@Path("service/{service}/tell/{path}")
	public Response tell(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;
	
	// synchronous service invocation, returns invocation result
	@POST
	@Path("service/{service}/call/{path}")
	@Retry(maxRetries = maxRetry)
	public Response call(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

	// asynchronous actor invocation, returns "OK" immediately
    @POST
	@Path("actor/{type}/{id}}/tell/{path}")
    @Retry(maxRetries = maxRetry)
	public Response actorTell(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    // synchronous actor invocation: returns invocation result
    @POST
   	@Path("actor/{type}/{id}}/call/{path}")
    @Retry(maxRetries = maxRetry)
   	public Response actorCall(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    // reminder operations
    @DELETE
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorCancelReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    @GET
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    @POST
   	@Path("actor/{type}/{id}/reminder}")
    @Retry(maxRetries = maxRetry)
    public Response actorScheduleReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    
    /*
     * Actor State Operations
     */
    @GET
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;
    
    @POST
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorSetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, Map<String,Object> params) throws ProcessingException;
    
    @DELETE
   	@Path("actor/{type}/{id}/state/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;
    
    @GET
   	@Path("actor/{type}/{id}/state")
    @Retry(maxRetries = maxRetry)
    public Response actorGetAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
    @DELETE
   	@Path("actor/{type}/{id}/state")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
   
    
	// broadcast to all sidecars except for ours
	@POST
	@Path("system/broadcast/${path}")
	@Retry(maxRetries = maxRetry)
	public Response broadcast(@PathParam("path") String path, Map<String,Object> params) throws ProcessingException;
   
}