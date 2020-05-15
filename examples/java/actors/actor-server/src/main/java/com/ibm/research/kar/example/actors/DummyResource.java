package com.ibm.research.kar.example.actors;

import java.util.logging.ConsoleHandler;
import java.util.logging.Logger;

import javax.enterprise.context.ApplicationScoped;
import javax.ws.rs.GET;
import javax.ws.rs.Path;

@Path("/dummy")
@ApplicationScoped
public class DummyResource {
	
	private static Logger logger = Logger.getLogger(DummyResource.class.getName());

	@GET
	@Path("hello")
	public String hello() {
		logger.addHandler(new ConsoleHandler());
		logger.info("Dummy hello says logger works!");
		System.out.println("stdout works too, Server sends greetings");
		return "Hello";
	}
}
