package com.ibm.research.kar.example.client;

import java.util.concurrent.CompletionStage;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonValue;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.eclipse.microprofile.rest.client.inject.RestClient;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.KarRest;

@Path("client")
@ApplicationScoped
@Consumes(MediaType.APPLICATION_JSON)
@Produces(MediaType.APPLICATION_JSON)
public class ClientResource {

	@Inject @ConfigProperty(name="useKar", defaultValue="true")
	boolean useKar;

	@Inject
	@RestClient
	private IncrServer defaultRestClient;

	@POST
	@Path("incrSync")
	public Response call(JsonValue num) throws ProcessingException {

		try {
			JsonValue result = Kar.call("number", "number/incr", num);
			Response resp = Response.status(Response.Status.OK).entity(result).build();
			return resp;
		} catch (Exception ex) {

			return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(ex.getMessage()).build();
		}

	}

	@POST
	@Path("incrAsync")
	public Response callAsync(JsonValue num) throws ProcessingException {

		try {
			CompletionStage<JsonValue> cf = Kar.callAsync("number", "number/incr", num);
			
			JsonValue value = cf
                    .toCompletableFuture()
                    .get();
			
			Response resp = Response.status(Response.Status.OK).entity(value).build();
			return resp;

		} catch (Exception ex) {

			return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(ex.getMessage()).build();
		}

	}

}
