package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.*;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.ExecutionException;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

/**
 * This sample actor mainly serves to illustrate some of the capabilities of
 * the actor-related APIs of the KAR Java SDK. It also shows some examples
 * of how to use javax.json APIs to manipulate and create JsonValues.
 */
@Actor
public class Sample extends ActorSkeleton {

	@Remote
	public JsonValue canBeInvoked(JsonObject arg) {
		System.out.println(this.toString() + ".canBeInvoked: My session id is " + this.getSession());
		return actorCall(this, this, "incr", arg);
	}

	@Remote
	public JsonValue incr(JsonObject json) {
		int number = json.getInt("number");
		return Json.createObjectBuilder().add("number", number + 1).build();
	}

	@Remote
	public JsonValue accumulate(JsonObject json) {
		int number = json.getInt("number");
		JsonValue result = actorCall(actorRef("calculator", this.getId()), "add", Json.createValue(number));
		return Json.createObjectBuilder().add("number", result).build();
	}

	@Remote
	public JsonValue incrFail(JsonObject json) {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		// intentionally invoke an undefined method to test exception behavior
		return actorCall(actorRef("calculator", this.getId()), "noOneAtHome", n);
	}

	@Remote
	public JsonValue asyncIncr(JsonObject json) throws InterruptedException, ExecutionException {
		int number = json.getInt("number");
		JsonValue n = Json.createValue(number);
		CompletionStage<JsonValue> cf = actorCallAsync(actorRef("calculator", this.getId()), "add", n);
		JsonValue result = cf.toCompletableFuture().get();
		return Json.createObjectBuilder().add("number", result).build();
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
	public JsonNumber setStateSubMap(JsonString key, JsonObject updates) {
		int numNew = actorSetMultipleState(this, key.getString(), updates);
		return Json.createValue(numNew);
	}

	@Remote
	public JsonNumber setStateSubkey(JsonString key, JsonString subkey, JsonValue value) {
		int numNew = actorSetState(this, key.getString(), subkey.getString(), value);
		return Json.createValue(numNew);
	}

	@Remote
	public JsonValue getStateSubkey(JsonString key, JsonString subkey) {
		return actorGetState(this, key.getString(), subkey.getString());
	}

	@Remote
	public JsonValue getStateElement(JsonString key) {
		return actorGetState(this, key.getString());
	}

	@Remote
	public JsonValue subMapSize(JsonString key) {
		return Json.createValue(actorSubMapSize(this, key.getString()));
	}

	@Remote
	public JsonValue subMapClear(JsonString key) {
		return Json.createValue(actorSubMapClear(this, key.getString()));
	}

	@Remote
	public JsonValue subMapGet(JsonString key) {
		Map<String,JsonValue> state = actorSubMapGet(this, key.getString());
		if (state instanceof JsonObject) {
			return (JsonObject)state;
		} else {
			JsonObjectBuilder builder = Json.createObjectBuilder();
			for (Entry<String,JsonValue> e: state.entrySet()) {
				builder.add(e.getKey(), e.getValue());
			}
			return builder.build();
		}
	}

	@Remote
	public void subMapPrintKeys(JsonString key) {
		String[] keys = actorSubMapKeys(this, key.getString());
		for (int i=0; i<keys.length; i++) {
			System.out.println(keys[i]);
		}
	}

	@Remote
	public JsonValue hasStateSubkey(JsonString key, JsonString subkey) {
		return actorContainsState(this, key.getString(), subkey.getString()) ? JsonValue.TRUE : JsonValue.FALSE;
	}

	@Remote
	public JsonValue hasStateElement(JsonString key) {
		return actorContainsState(this, key.getString()) ? JsonValue.TRUE : JsonValue.FALSE;
	}

	@Remote
	public JsonObject getState() {
		Map<String,JsonValue> state = actorGetAllState(this);
		if (state instanceof JsonObject) {
			return (JsonObject)state;
		} else {
			JsonObjectBuilder builder = Json.createObjectBuilder();
			for (Entry<String,JsonValue> e: state.entrySet()) {
				builder.add(e.getKey(), e.getValue());
			}
			return builder.build();
		}
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

	@Remote void cancelReminder() {
		actorCancelReminder(this, "echo");
	}

	@Remote
	public void dumpReminders() {
		System.out.println(actorGetAllReminders(this)[0]);
	}
}
