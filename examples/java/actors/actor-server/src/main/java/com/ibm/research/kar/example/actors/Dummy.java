package com.ibm.research.kar.example.actors;

import javax.json.Json;

import javax.json.JsonObject;
import javax.json.JsonValue;
import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.KarSessionListener;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.LockPolicy;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy implements KarSessionListener {

	Kar kar = new Kar();

	private String sessionid;

	@Activate
	public void init() {

	}

	@Remote(lockPolicy = LockPolicy.READ)
	public JsonValue canBeInvoked(JsonObject json) {
		int number = json.getInt("number");
		number++;

		JsonObject params = Json.createObjectBuilder()
				.add("number",number)
				.build();

		JsonValue result = kar.actorCall("dummy2", "dummy2id", "canBeInvoked", params);

		System.out.println("Dummy.canBeInvoked: My session id is " + this.sessionid);
		return result;
	}

	@Remote(lockPolicy = LockPolicy.READ)
	public JsonValue incr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		return kar.actorCall("calculator", "mycalc", "add", n);
	}

	public void cannotBeInvoked() {
	}

	@Deactivate
	public void kill() {
	}

	@Override
	public void setSessionId(String sessionId) {
		this.sessionid = sessionId;
	}

	@Override
	public String getSessionId() {
		return this.sessionid;
	}
}
