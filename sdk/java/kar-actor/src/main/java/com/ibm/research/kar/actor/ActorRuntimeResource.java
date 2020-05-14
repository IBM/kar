package com.ibm.research.kar.actor;

import java.lang.reflect.Type;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;

import javax.annotation.Resource;
import javax.enterprise.concurrent.ManagedExecutorService;
import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import javax.json.JsonObject;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

@Path("/actor")
@ApplicationScoped
public class ActorRuntimeResource {

	private final static String LOG_PREFIX = "ActorRuntimResource";
	private final static int FUTURE_WAIT_TIME_MILLIS = 300;

	@Inject
	ActorManager actorManager;

	@Resource
	ManagedExecutorService managedExecutorService;

	@GET
	@Path("{type}/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getActor(@PathParam("type") String type, @PathParam("id") String id) {
		System.out.println(LOG_PREFIX + "getActor: Checking for actor with id " + id);
		if (actorManager.getActor(type, id) != null) {

			System.out.println(LOG_PREFIX + "getActor: Found actor");
			return Response.status(Response.Status.OK).build();
		} else {

			System.out.println(LOG_PREFIX + "getActor: No actor found, creating");

			this.actorManager.createActor(type, id);

			return Response.status(Response.Status.OK).entity("Created " + type + " actor " + id).build();
		}
	}

	@DELETE
	@Path("{type}/{id}")
	public Response deleteActor(@PathParam("type") String type, @PathParam("id") String id) {
		System.out.println(LOG_PREFIX + "deleteActor: deleting actor");

		//actorManager.deleteActor(type, id);
		return Response.status(Response.Status.OK).build();
	}

	@POST
	@Path("{type}/{id}/{sessionid}/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response invokeActorMethod(
			@PathParam("type") String type, 
			@PathParam("id") String id, 
			@PathParam("sessionid") String sessionid, 
			@PathParam("path") String path, 
			JsonObject params) {

		System.out.println(LOG_PREFIX + "invokeActorMethod: invoking " + type + "actor " + id + " method " + path + " with params " + params);

		Object actorObj = this.actorManager.getActor(type, id);
		RemoteMethodType methodType = this.actorManager.getActorMethod(type, path);

		System.out.println(LOG_PREFIX + "invokeActorMethod: actorObj is " + actorObj + " and method is " + methodType);
		Object result = null;


		if ((actorObj != null) && (methodType != null)) {
			Type[] interfaces = actorObj.getClass().getGenericInterfaces();

			// give object sessionId if it is listening for it
			for (Type t: interfaces) {
				if (t.getTypeName() == KarSessionListener.class.getTypeName()) {
					synchronized(actorObj) {
						((KarSessionListener)actorObj).setSessionId(sessionid);
					}
				}
			}

			ActorTask task = new ActorTask();
			task.setActorObj(actorObj);
			task.setActorMethod(methodType.getMethod());
			task.setLockPolicy(methodType.getLockPolicy());
			task.setParams(params);



			// execute task asynchronously
			Future<Object> futureResult = managedExecutorService.submit(task);

			try {
				while (!futureResult.isDone()) {
					Thread.sleep(FUTURE_WAIT_TIME_MILLIS);
				} 

				result = futureResult.get();


			} catch (InterruptedException e) {
				e.printStackTrace();
				System.out.println(LOG_PREFIX + "invokeActorMethod: waiting interrupted");
			} catch (ExecutionException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
				System.out.println(LOG_PREFIX + "invokeActorMethod: execution error for actor method");
			}

		} else {
			System.out.println(LOG_PREFIX+"invokeActorMethod: Warning, cannot find " + type + " actor instance " + id + " or method " + path);
			return Response.status(Response.Status.NOT_FOUND).build();
		}

		if (result == null) {
			Response.status(Response.Status.OK).build();
		} 

		return Response.status(Response.Status.OK).entity(result).build();

	}

}
