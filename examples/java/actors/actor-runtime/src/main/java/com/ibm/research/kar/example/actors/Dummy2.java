package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonObject;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy2 {

	@Activate
	public void init() {
		
	}
	
	@Remote
	public String canBeInvoked() {
		JsonObject params = Json.createObjectBuilder()
				.add("number",100)
				.build();
		
		return params.toString();
	}
	
	public void cannotBeInvoked() {
		
	}
	
	@Deactivate
	public void kill() {
		
	}
}
