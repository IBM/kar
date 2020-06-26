package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.*;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Map.Entry;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorRef;

import java.util.concurrent.CompletionStage;
import java.util.concurrent.ExecutionException;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;
import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;

@Actor
public class Dummy extends ActorBoilerplate {

	@Activate
	public void init() {
	}

	@Deactivate
	public void kill() {
	}

	@Remote
	public JsonValue canBeInvoked(JsonObject json) {
		int number = json.getInt("number");
		number++;

		JsonObject params = Json.createObjectBuilder().add("number", number).build();
		ActorRef dummy2 = actorRef("dummy2", "dummy2id");
		JsonValue result = actorCall(dummy2, "canBeInvoked", params);

		System.out.println("Dummy.canBeInvoked: My session id is " + this.session);
		return result;
	}

	@Remote
	public JsonValue incr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		return actorCall(actorRef("calculator", "mycalc"), "add", n);
	}

	@Remote
	public JsonValue incrFail(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		return actorCall(actorRef("calculator", "mycalc"), "magic", n); // intentionally invoke an undefined method to test exception behavior
	}

	@Remote
	public JsonValue asyncIncr(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		try {
			CompletionStage<JsonValue> cf = actorCallAsync(actorRef("calculator", "mycalc"), "add", n);

			return cf.exceptionally(t -> {
				// exception thrown
				System.out.println("In exceptionally");
				if (t instanceof ActorMethodNotFoundException) {
					System.out.println(t.getCause());
				}
				return Json.createValue("Exceptionally: Error invoking asyncIncr");
			}).toCompletableFuture().get();
		} catch (InterruptedException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (ExecutionException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}

		System.out.println("Never got here");

		return Json.createValue("Error invoking asyncIncr");
	}

	@Remote
	public void echo(JsonString msg) {
		System.out.println(msg);
	}

	@Remote
	public void echoFriend() {
		actorCall(this, this, "echo", Json.createValue("Hello Friend"));
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
		Map<String, JsonValue> state = actorGetAllState(this);
		for (Entry<String, JsonValue> e : state.entrySet()) {
			System.out.println(e.getKey() + " = " + e.getValue().toString());
		}
		return Json.createValue(state.size());
	}

	@Remote
	public void createReminder(JsonString msg) {
		actorScheduleReminder(this, "echo", "r1", Instant.now().plusSeconds(1), Duration.ofSeconds(5), msg);
	}

	@Remote
	public void dumpReminders() {
		System.out.println(actorGetAllReminders(this)[0]);
	}
}
