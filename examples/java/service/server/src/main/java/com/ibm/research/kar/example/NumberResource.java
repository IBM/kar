package com.ibm.research.kar.example;

import javax.json.JsonNumber;
import javax.json.JsonValue;
import javax.json.spi.JsonProvider;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

@Path("number")
@Consumes(MediaType.APPLICATION_JSON)
@Produces(MediaType.APPLICATION_JSON)
public class NumberResource {

	/**
	 * Reads num value, increments and returns new value
	 *
	 * @param num
	 * @return num++ or error
	 */
	@POST
	@Path("incr")
	public JsonValue incrNumber(JsonValue num) {
		int input = ((JsonNumber) num).intValue();
		return JsonProvider.provider().createValue(input + 1);
	}
}
