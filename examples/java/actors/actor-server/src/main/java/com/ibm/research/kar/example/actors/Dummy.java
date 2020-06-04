package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.ActorException;
import com.ibm.research.kar.ActorMethodNotFoundException;
import com.ibm.research.kar.actor.ActorRef;
import static com.ibm.research.kar.Kar.*;

import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.ExecutionException;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Dummy extends ActorBoilerplate {

	@Activate
	public void init() {
	}

	@Remote
	public JsonValue canBeInvoked(JsonObject json) {
		int number = json.getInt("number");
		number++;

		JsonObject params = Json.createObjectBuilder()
				.add("number",number)
				.build();

		ActorRef dummy2 = actorRef("dummy2", "dummy2id");
		JsonValue result = null;
		try {
			result = actorCall(dummy2, "canBeInvoked", params);
		} catch (ActorMethodNotFoundException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}

		System.out.println("Dummy.canBeInvoked: My session id is " + this.session);
		return result;
	}

	@Remote
	public JsonValue incr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		try {
			return actorCall(actorRef("calculator", "mycalc"), "add", n);
		} catch (ActorMethodNotFoundException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
		
		return null;
	}

	public void cannotBeInvoked() {
	}

	@Remote
	public JsonValue asyncIncr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		try {
			CompletionStage<JsonValue> cf = actorCallAsync(actorRef("calculator", "mycalc"), "add", n);


			return cf
						.exceptionally(t -> {
						// exception thrown
						System.out.println("In exceptionally");
						if (t instanceof ActorMethodNotFoundException) {
							System.out.println(t.getCause());
						}
						return Json.createValue("Exceptionally: Error invoking asyncIncr");
					})
					.toCompletableFuture()
					.get();
		} catch (InterruptedException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (ExecutionException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (ActorMethodNotFoundException e) {
			System.out.println("In actor exception");
			e.printStackTrace();
		}

		System.out.println("Never got here");

		return Json.createValue("Error invoking asyncIncr");
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
