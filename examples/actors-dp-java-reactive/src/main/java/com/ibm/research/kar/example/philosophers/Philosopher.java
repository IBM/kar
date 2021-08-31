/*
 * Copyright IBM Corporation 2020,2021
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.ibm.research.kar.example.philosophers;

import java.time.Instant;
import java.util.Map;
import java.util.UUID;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
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
		Map<String, JsonValue> state = Actors.State.getAll(this);
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
		Actors.State.set(this, state);
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
		Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step);
		this.step = step;
		checkpointState();
	}

	@Remote
	public void getFirstFork(JsonNumber attempt, JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (Actors.call(Actors.ref("Fork", this.firstFork.getString()), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			Actors.tell(this, "getSecondFork", Json.createValue(1), step);
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to acquire his first Fork "+attempt+" times");
			}
			Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1), step);
		}
		this.step = step;
		Actors.State.set(this, "step", step);
	}

	@Remote
	public void getSecondFork(JsonNumber attempt, JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (Actors.call(Actors.ref("Fork", this.secondFork.getString()), "pickUp", Json.createValue(this.getId())).equals(JsonValue.TRUE)) {
			Actors.tell(this, "eat", step);
		} else {
			if (attempt.intValue() > 5) {
				System.out.println("Warning: "+this.getId()+" has failed to acquire his second Fork "+attempt+" times");
			}
			Actors.Reminders.schedule(this, "getSecondFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue()+1), step);
		}
		this.step = step;
		Actors.State.set(this, "step", step);
	}

	@Remote
	public void eat(JsonString step) {
		if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
		step = Json.createValue(UUID.randomUUID().toString());
		if (VERBOSE) System.out.println(this.getId()+" ate serving number "+this.servingsEaten);
		Actors.call(Actors.ref("Fork", this.secondFork.getString()), "putDown", Json.createValue(this.getId()));
		Actors.call(Actors.ref("Fork", this.firstFork.getString()), "putDown", Json.createValue(this.getId()));
		if (this.servingsEaten.intValue() < this.targetServings.intValue()) {
			Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step);
		} else {
			Actors.call(Actors.ref("Table", this.table.getString()), "doneEating", Json.createValue(this.getId()));
		}
		this.servingsEaten = Json.createValue(this.servingsEaten.intValue() + 1);
		this.step = step;
		checkpointState();
	}
}
