package com.ibm.research.kar.actor.tests;

import static org.junit.jupiter.api.Assertions.assertNotNull;

import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import org.junit.jupiter.api.Test;

import com.ibm.research.kar.Kar;

public class ActorTest {
	
	@Test
	void testActorCall() {
		Kar kar = new Kar();
		
		JsonObject params = Json.createObjectBuilder()
				.add("number", 1)
				.build();
		
		Response resp = kar.actorCall("dummy", "dummyid", "canBeInvoked", params);
		assertNotNull(resp);
	}

}
