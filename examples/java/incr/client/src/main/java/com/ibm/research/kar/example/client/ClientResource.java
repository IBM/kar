package com.ibm.research.kar.example.client;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
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

@Path("client")
@ApplicationScoped
@Consumes(MediaType.APPLICATION_JSON)
@Produces(MediaType.APPLICATION_JSON)
public class ClientResource {

	@Inject @ConfigProperty(name="useKar",
			 defaultValue="true")
	boolean useKar;

	@Inject
	private Kar karClient;

	@Inject
	@RestClient
	private IncrServer defaultRestClient;


	@POST
	@Path("incrSync")
	public Response call(KarParams karParams) throws ProcessingException {

		try {

			if (useKar == true) {
				JsonValue result = karClient.call(karParams.service, karParams.path, karParams.params);
				Response resp = Response.status(Response.Status.OK).entity(result).build();
				return resp;
			} else {
				JsonNumber bg = (JsonNumber)karParams.params.get("number");
				int num = bg.intValue();

				Number numObj = new Number();
				numObj.setNumber(num);

				Number respNum = this.defaultRestClient.incrNumber(numObj);

				Response resp = Response.status(Response.Status.OK).entity(respNum).build();
				return resp;
			}
		} catch (Exception ex) {

			return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(ex.getMessage()).build();
		}

	}

	@POST
	@Path("incrAsync")
	public Response tell(KarParams karParams) throws ProcessingException {
		try {
			if (useKar == true) {
				this.karClient.tell(karParams.service, karParams.path, karParams.params);
				return Response.status(Response.Status.OK).build();
			} else {
				return Response.status(Response.Status.OK).build();
			}
		} catch (Exception ex) {
			return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(ex).build();
		}
	}

}
