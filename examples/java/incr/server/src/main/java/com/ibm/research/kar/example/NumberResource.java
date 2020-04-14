package com.ibm.research.kar.example;

import javax.enterprise.context.ApplicationScoped;
import javax.ws.rs.Consumes;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;


@Path("/number")
@ApplicationScoped
public class NumberResource {

	NumberService numService = new NumberService();
	/**
	 * Reads num value, increments and returns new value
	 * @param num
	 * @return num++ or error
	 */
	@POST
	@Path("/incr")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Number incrNumber(Number num) {

		int oldNum = num.getNumber();
		num.setNumber(numService.incr(oldNum));

		return num;

	}

	/**
	 * @return the answer to the Ultimate Question of Life, the Universe, and Everything
	 */
	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Number getNumber() {
		return numService.getNum();
	}

}
