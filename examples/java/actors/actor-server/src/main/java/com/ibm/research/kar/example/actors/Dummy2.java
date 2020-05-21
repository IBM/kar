package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonObject;

import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy2 extends ActorBoilerplate {

	@Remote
	public JsonObject canBeInvoked(JsonObject json) {
		int number = json.getInt("number");
		number++;

		JsonObject params = Json.createObjectBuilder()
				.add("number", number)
				.build();	
	
		//uncomment to test long running
		/*
		try {
			Thread.sleep(10000);
		} catch (InterruptedException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} */

		return params;
	}
}
