package com.ibm.research.kar.example;

import java.math.BigDecimal;
import java.util.Map;

import javax.enterprise.context.ApplicationScoped;
import javax.ws.rs.Consumes;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

@Path("/kar/v1")
@ApplicationScoped
public class KarResource {

	NumberService numService = new NumberService();

	@POST
	@Path("/service/{service}/tell/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response tell(Map<String,Object> params,
			@PathParam("service") String service,
			@PathParam("path") String path) {

		try {

			if (service.contentEquals("number") && path.contentEquals("incr")) {

				return Response
						.status(Response.Status.OK)
						.entity("TELL: " + service + " at path " + path + " with params " + params)
						.build();
			} else {

				return Response
						.status(Response.Status.OK)
						.entity("TELL: " + service + " at path " + path + " with params " + params)
						.build();
			}
		} catch (Exception ex) {
			return Response
					.status(Response.Status.INTERNAL_SERVER_ERROR)
					.entity(ex)
					.build();
		}

	}

	@POST
	@Path("/service/{service}/call/{path}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response call(Map<String,Object> params,
			@PathParam("service") String service,
			@PathParam("path") String path) {

		try {

			if (service.contentEquals("number") && path.contentEquals("incr")) {
				
				BigDecimal bg = (BigDecimal)params.get("number");
				int num = bg.intValue();

				Number number = new Number();
				number.setNumber(numService.incr(num));

				return Response
						.status(Response.Status.OK)
						.entity(number)
						.build();
			} else {

				return Response
						.status(Response.Status.OK)
						.entity("CALL: " + service + " at path " + path + " with params " + params)
						.build();
			}
		} catch (Exception ex) {
			return Response
					.status(Response.Status.INTERNAL_SERVER_ERROR)
					.entity(ex)
					.build();
		}
	}

	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getKar() {

		return Response
				.status(Response.Status.OK)
				.entity("Hello from KAR server!")
				.build();
	}


}
