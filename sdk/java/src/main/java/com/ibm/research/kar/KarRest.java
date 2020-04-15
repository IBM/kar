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
@Path("kar")
public interface KarRest extends AutoCloseable {
	
	int maxRetry = 10;
	
	/*
	 * Public methods
	 */
	
	// asynchronous service invocation, returns "OK" immediately
	@POST
	@Path("tell/{service}/{path}")
	public Response tell(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;
	
	// synchronous service invocation, returns invocation result
	@POST
	@Path("call/{service}/{path}")
	@Retry(maxRetries = maxRetry)
	public Response call(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

	// asynchronous actor invocation, returns "OK" immediately
    @POST
	@Path("actor-tell/{type}/{id}}/{path}")
    @Retry(maxRetries = maxRetry)
	public Response actorTell(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    // synchronous actor invocation: returns invocation result
    @POST
   	@Path("actor-call/{type}/{id}}/{path}")
    @Retry(maxRetries = maxRetry)
   	public Response actorCall(@PathParam("type") String type, @PathParam("id") String id, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    // reminder operations
    @POST
   	@Path("actor-reminder/{type}/{id}/cancel}")
    @Retry(maxRetries = maxRetry)
    public Response actorCancelReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    @POST
   	@Path("actor-reminder/{type}/{id}/get}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    @POST
   	@Path("actor-reminder/{type}/{id}/schedule}")
    @Retry(maxRetries = maxRetry)
    public Response actorScheduleReminder(@PathParam("type") String type, @PathParam("id") String id, Map<String,Object> params) throws ProcessingException;
    
    
    /*
     * Actor State Operations
     */
    @GET
   	@Path("actor-state/{type}/{id}/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;
    
    @POST
   	@Path("actor-state/{type}/{id}/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorSetState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key, Map<String,Object> params) throws ProcessingException;
    
    @DELETE
   	@Path("actor-state/{type}/{id}/{key}")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteState(@PathParam("type") String type, @PathParam("id") String id, @PathParam("key") String key) throws ProcessingException;
    
    @GET
   	@Path("actor-state/{type}/{id}")
    @Retry(maxRetries = maxRetry)
    public Response actorGetAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
    @DELETE
   	@Path("actor-state/{type}/{id}")
    @Retry(maxRetries = maxRetry)
    public Response actorDeleteAllState(@PathParam("type") String type, @PathParam("id") String id) throws ProcessingException;
    
   
    
	// broadcast to all sidecars except for ours
	@POST
	@Path("broadcast/${path}")
	@Retry(maxRetries = maxRetry)
	public Response broadcast(@PathParam("path") String path, Map<String,Object> params) throws ProcessingException;
   
}