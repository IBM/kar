package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy {
	
	Kar kar = new Kar();

	@Activate
	public void init() {
		
	}
	
	@Remote
	public String canBeInvoked() {
		
		JsonObject params = Json.createObjectBuilder()
				.add("number",205)
				.build();
		
		Response resp = kar.actorCall("dummy2", "dummy2id", "canBeInvoked", params);
	
		JsonObject respObj = resp.readEntity(JsonObject.class);
		return respObj.toString();
	}
	
	public void cannotBeInvoked() {
		
	}
	
	@Deactivate
	public void kill() {
		
	}
}
