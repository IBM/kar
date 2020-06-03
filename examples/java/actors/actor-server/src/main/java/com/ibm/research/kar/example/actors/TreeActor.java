package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorRef;
import static com.ibm.research.kar.Kar.*;

import java.util.Map;
import java.util.Map.Entry;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class TreeActor extends ActorBoilerplate {

	@Activate
	public void init() {
	}
	
	@Remote
	public JsonValue callTree(JsonObject json) {
		String label = json.getString("label","top");
		int level = json.getInt("level",1);
		int maxdepth = json.getInt("maxdepth");
		boolean trace = json.getBoolean("trace", false);
		int replies = 0;
		long starttime = 0;

		if ( 1 == level) {
			starttime = System.nanoTime();
		}

		if (trace)
			System.out.println("TreeActor:" + this.getId() + " Received level " + level);

		if (level >= maxdepth) {
			JsonObject params = Json.createObjectBuilder()
					.add("replies", 0)
					.build();
			return params;
		}
		
		JsonObject paramsA = Json.createObjectBuilder()
				.add("label",  label+level+"A")
				.add("level", level+1)
				.add("maxdepth", maxdepth)
				.add("trace", trace)
				.build();

		ActorRef actorA = actorRef("tree",label+level+"A");
		ActorRef actorB = actorRef("tree",label+level+"B");

		JsonValue resultA = actorCall(actorA, "callTree", paramsA);
		replies = 1 + resultA.asJsonObject().getInt("replies");

		JsonObject paramsB = Json.createObjectBuilder()
				.add("label",  label+level+"B")
				.add("level", level+1)
				.add("maxdepth", maxdepth)
				.add("trace", trace)
				.build();
		JsonValue resultB = actorCall(actorB, "callTree", paramsB);
		replies += 1 + resultB.asJsonObject().getInt("replies");

		long duration = 0;
		if ( 1 == level) {
			duration = (System.nanoTime()-starttime)/1000000;
			System.out.println("recursion took " + duration + "ms");
		}

		JsonObject result = Json.createObjectBuilder()
				.add("replies", replies)
				.add("duration", duration)
				.build();

		return result;
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