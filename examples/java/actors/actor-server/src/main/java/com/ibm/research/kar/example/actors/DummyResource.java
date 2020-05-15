package com.ibm.research.kar.example.actors;

import javax.enterprise.context.ApplicationScoped;
import javax.ws.rs.GET;
import javax.ws.rs.Path;

@Path("/dummy")
@ApplicationScoped
public class DummyResource {

	@GET
	@Path("hello")
	public String hello() {
		System.out.println("Server sends greetings");
		return "Hello";
	}
}
