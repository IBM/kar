package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorGetAllState;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorScheduleReminder;
import static com.ibm.research.kar.Kar.actorSetMultipleState;
import static com.ibm.research.kar.Kar.actorSetState;
import static com.ibm.research.kar.Kar.actorTell;

import java.time.Instant;
import java.util.Map;
import java.util.UUID;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Philosopher extends ActorSkeleton {
	private JsonString table;
	private JsonString firstFork;
	private JsonString secondFork;
	private JsonNumber servingsEaten;
	private JsonNumber targetServings;
	private JsonString step;

	static final boolean VERBOSE = Boolean.parseBoolean(System.getenv("VERBOSE"));

	@Activate
	public void activate () {
		Map<String, JsonValue> state = actorGetAllState(this);
		if (state.containsKey("table")) {
			this.table = ((JsonString)state.get("table"));
		}
		if (state.containsKey("firstFork")) {
			this.firstFork = ((JsonString)state.get("firstFork"));
		}
		if (state.containsKey("secondFork")) {
			this.secondFork = ((JsonString)state.get("secondFork"));
		}
		if (state.containsKey("servingsEaten")) {
			this.servingsEaten = ((JsonNumber)state.get("servingsEaten"));
		}
		if (state.containsKey("targetServings")) {
			this.targetServings = ((JsonNumber)state.get("targetServings"));
		}
		if (state.containsKey("step")) {
			this.step = ((JsonString)state.get("step"));
		} else {
			// Initial step for an uninitialized Philosopher is its id
			this.step = Json.createValue(this.getId());
		}
	}

	private void checkpointState() {
		JsonObjectBuilder jb = Json.createObjectBuilder();
		jb.add("table", this.table);
		jb.add("firstFork", this.firstFork);
		jb.add("secondFork", this.secondFork);
		jb.add("servingsEaten", this.servingsEaten);
		jb.add("targetServings", this.targetServings);
		jb.add("step", this.step);
		JsonObject state = jb.build();
		actorSetMultipleState(this, state);
	}

	private Instant nextStepTime() {
		int thinkTime = (int)(Math.random() * 1000); // random 0...999ms
		return Instant.now().plusMillis(thinkTime);
	}

	@Remote
	public void joinTable(JsonString table, JsonString firstFork, JsonString secondFork, JsonNumber targetServings, JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		this.table = table;
		this.firstFork = firstFork;
		this.secondFork = secondFork;
		this.servingsEaten = Json.createValue(0);
		this.targetServings = targetServings;
		actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step);
		this.step = step;
		checkpointState();
	}

	@Remote
	public void getFirstFork(JsonNumber attempt, JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (actorCall(actorRef("Fork", this.firstFork.getString()), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			actorTell(this, "getSecondFork", Json.createValue(1), step);
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to aqquire his first Fork "+attempt+" times");
			}
			actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1), step);
		}
		this.step = step;
		actorSetState(this, "step", step);
	}

	@Remote
	public void getSecondFork(JsonNumber attempt, JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (actorCall(actorRef("Fork", this.secondFork.getString()), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			actorTell(this, "eat", step);
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to aqquire his second Fork "+attempt+" times");
			}
			actorScheduleReminder(this, "getSecondFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1), step);
		}
		this.step = step;
		actorSetState(this, "step", step);
	}

	@Remote
	public void eat(JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (VERBOSE) System.out.println(this.getId()+" ate serving number "+this.servingsEaten);
		actorCall(actorRef("Fork", this.secondFork.getString()), "putDown", Json.createValue(this.getId()));
		actorCall(actorRef("Fork", this.firstFork.getString()), "putDown", Json.createValue(this.getId()));
		if (this.servingsEaten.intValue() < this.targetServings.intValue()) {
			actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step);
		} else {
			actorCall(actorRef("Table", this.table.getString()), "doneEating", Json.createValue(this.getId()));
		}
		this.servingsEaten = Json.createValue(this.servingsEaten.intValue() + 1);
		this.step = step;
		checkpointState();
	}
}
