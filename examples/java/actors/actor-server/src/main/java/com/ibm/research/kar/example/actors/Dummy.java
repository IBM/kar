package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.ActorRef;
import static com.ibm.research.kar.Kar.*;

import java.util.Map;
import java.util.Map.Entry;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.LockPolicy;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy extends ActorBoilerplate {

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

		ActorRef dummy2 = actorRef("dummy2", "dummy2id");
		JsonValue result = actorCall(dummy2, "canBeInvoked", params);

		System.out.println("Dummy.canBeInvoked: My session id is " + this.session);
		return result;
	}

	@Remote(lockPolicy = LockPolicy.READ)
	public JsonValue incr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		return actorCall(actorRef("calculator", "mycalc"), "add", n);
	}

	public void cannotBeInvoked() {
	}

	@Remote
	public JsonNumber setState(JsonObject updates) {
		int numNew = actorSetMultipleState(this, updates);
		return Json.createValue(numNew);
	}

	@Remote
	public JsonValue getStateElement(JsonString key) {
		return actorGetState(this, key.getString());
	}

	@Remote
	public JsonNumber printState() {
		Map<String,JsonValue> state = actorGetAllState(this);
		for (Entry<String,JsonValue> e: state.entrySet()) {
			System.out.println(e.getKey() + " = " + e.getValue().toString());
		}
		return Json.createValue(state.size());
	}

	@Deactivate
	public void kill() {
	}
}
