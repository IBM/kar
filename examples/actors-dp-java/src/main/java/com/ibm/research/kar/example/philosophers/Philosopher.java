package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorGetAllState;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorScheduleReminder;
import static com.ibm.research.kar.Kar.actorSetMultipleState;
import static com.ibm.research.kar.Kar.actorTell;

import java.time.Instant;
import java.util.Map;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Philosopher extends ActorSkeleton {
	private String cafe;
	private String firstFork;
	private String secondFork;
	private int servingsEaten;
	private int targetServings;

	@Activate
	public void activate () {
		Map<String, JsonValue> state = actorGetAllState(this);
		if (state.containsKey("cafe")) {
			this.cafe = ((JsonString)state.get("cafe")).getString();
		}
		if (state.containsKey("firstFork")) {
			this.firstFork = ((JsonString)state.get("firstFork")).toString();
		}
		if (state.containsKey("secondFork")) {
			this.secondFork = ((JsonString)state.get("secondFork")).toString();
		}
		if (state.containsKey("servingsEaten")) {
			this.servingsEaten = ((JsonNumber)state.get("servingsEaten")).intValue();
		}
		if (state.containsKey("targetServings")) {
			this.targetServings = ((JsonNumber)state.get("targetServings")).intValue();
		}
	}

	@Deactivate
	public void deactivate() {
		this.checkpointState();
	}

	private void checkpointState() {
		JsonObjectBuilder jb = Json.createObjectBuilder();
		jb.add("cafe", Json.createValue(this.cafe));
		jb.add("firstFork", Json.createValue(this.firstFork));
		jb.add("secondFork", Json.createValue(this.secondFork));
		jb.add("servingsEaten", Json.createValue(this.servingsEaten));
		jb.add("targetServings", Json.createValue(this.targetServings));
		JsonObject state = jb.build();
		actorSetMultipleState(this, state);
	}

	private Instant nextStepTime() {
		int thinkTime = (int)(Math.random() * 1000);
		return Instant.now().plusMillis(thinkTime);
	}

	@Remote
	public void joinTable(JsonString cafe, JsonString firstFork, JsonString secondFork, JsonNumber targetServings) {
		System.out.println("start join table"+this.getId());
		this.cafe = cafe.getString();
		this.firstFork = firstFork.getString();
		this.secondFork = secondFork.getString();
		this.servingsEaten = 0;
		this.targetServings = targetServings.intValue();
		this.checkpointState();
		actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1));
		System.out.println("end join table"+this.getId());
	}

	@Remote
	public void getFirstFork(JsonNumber attempt) {
		if (actorCall(actorRef("Fork", this.firstFork), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			actorTell(this, "getSecondFork", Json.createValue(1));
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to aqquire his first Fork "+attempt+" times");
			}
			actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1));
		}
	}

	@Remote
	public void getSecondFork(JsonNumber attempt) {
		if (actorCall(actorRef("Fork", this.secondFork), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			actorTell(this, "eat", Json.createValue(this.servingsEaten));
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to aqquire his second Fork "+attempt+" times");
			}
			actorScheduleReminder(this, "getSecondFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1));
		}
	}

	@Remote
	public void eat(JsonNumber servingsEaten) {
		System.out.println(this.getId()+" ate serving number "+servingsEaten);
		this.servingsEaten = servingsEaten.intValue() + 1;
		this.checkpointState();
		actorCall(actorRef("Fork", this.secondFork), "putDown", Json.createValue(this.getId()));
		actorCall(actorRef("Fork", this.firstFork), "putDown", Json.createValue(this.getId()));
		if (this.servingsEaten < this.targetServings) {
			actorScheduleReminder(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1));
		} else {
			actorCall(actorRef("Cafe", this.cafe), "doneEating", Json.createValue(this.getId()));
		}
	}
}
