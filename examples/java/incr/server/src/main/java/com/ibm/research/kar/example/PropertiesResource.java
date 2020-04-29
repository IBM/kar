package com.ibm.research.kar.example;

import java.util.Properties;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;


/**
 * Get system properties as a test
 *
 */
@Path("properties")
public class PropertiesResource {

	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Properties getProperties() {
		return System.getProperties();
		
	}
}
