package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.actorGetAllState;
import static com.ibm.research.kar.Kar.actorGetState;
import static com.ibm.research.kar.Kar.actorSetMultipleState;

import java.util.Map;
import java.util.Map.Entry;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class TreeActorCounter extends ActorBoilerplate {

	int expectedLeaves;
	long starttime;
	long nextreport;
	
	@Activate
	public void init() {
	}
	
	@Remote
	public void setCount(JsonObject json) {
		expectedLeaves = json.getInt("expected");

		starttime = System.currentTimeMillis();
		nextreport = starttime + 1000;
		System.out.println("TreeCounter:setCount expecting " + expectedLeaves + " notifies");
	}

	@Remote
	public void callDone(JsonObject json) {
		String leaf = json.getString("leaf");
		boolean trace = json.getBoolean("trace", false);

		--expectedLeaves;
		if (0 == expectedLeaves) {
			System.out.println("TreeCounter:callDone notified by " + leaf + " no more expected ");
			return;
		}
		if (trace && 0 < expectedLeaves) {
			System.out.println("TreeCounter:callDone notified by " + leaf + " still expecting " + expectedLeaves);
			return;
		}
		long currenttime = System.currentTimeMillis();
		if (currenttime >= nextreport) {
			System.out.println("TreeCounter:callDone notified by " + leaf + " still expecting " + expectedLeaves);
			nextreport = currenttime + 1000;
		}
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
