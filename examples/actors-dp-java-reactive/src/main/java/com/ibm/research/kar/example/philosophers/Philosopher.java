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

import io.smallrye.mutiny.Uni;

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
	public Uni<Void> activate() {
		return Actors.State.getAll(this).chain(state -> {
			if (state.containsKey("table")) {
				this.table = ((JsonString) state.get("table"));
			}
			if (state.containsKey("firstFork")) {
				this.firstFork = ((JsonString) state.get("firstFork"));
			}
			if (state.containsKey("secondFork")) {
				this.secondFork = ((JsonString) state.get("secondFork"));
			}
			if (state.containsKey("servingsEaten")) {
				this.servingsEaten = ((JsonNumber) state.get("servingsEaten"));
			}
			if (state.containsKey("targetServings")) {
				this.targetServings = ((JsonNumber) state.get("targetServings"));
			}
			if (state.containsKey("step")) {
				this.step = ((JsonString) state.get("step"));
			} else {
				// Initial step for an uninitialized Philosopher is its id
				this.step = Json.createValue(this.getId());
			}
			return Uni.createFrom().nullItem();
		});
	}

	private Uni<Void> checkpointState() {
		JsonObjectBuilder jb = Json.createObjectBuilder();
		jb.add("table", this.table);
		jb.add("firstFork", this.firstFork);
		jb.add("secondFork", this.secondFork);
		jb.add("servingsEaten", this.servingsEaten);
		jb.add("targetServings", this.targetServings);
		jb.add("step", this.step);
		JsonObject state = jb.build();
		return Actors.State.set(this, state);
	}

	private Instant nextStepTime() {
		int thinkTime = (int) (Math.random() * 1000); // random 0...999ms
		return Instant.now().plusMillis(thinkTime);
	}

	@Remote
	public Uni<Void> joinTable(JsonString table, JsonString firstFork, JsonString secondFork, JsonNumber targetServings, JsonString currentStep) {
		if (!this.step.equals(currentStep)) {
			return Uni.createFrom().failure(new RuntimeException("unexpected step"));
		}
		final JsonString step = Json.createValue(UUID.randomUUID().toString());
		this.table = table;
		this.firstFork = firstFork;
		this.secondFork = secondFork;
		this.servingsEaten = Json.createValue(0);
		this.targetServings = targetServings;
		return Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step)
			.chain(() -> {
				this.step = step;
				return checkpointState();
			});
	}

	@Remote
	public Uni<Void> getFirstFork(JsonNumber attempt, JsonString currentStep) {
		if (!this.step.equals(currentStep)) {
			return Uni.createFrom().failure(new RuntimeException("unexpected step"));
		}
		final JsonString step = Json.createValue(UUID.randomUUID().toString());
		return Actors.call(Actors.ref("Fork", this.firstFork.getString()), "pickUp", Json.createValue(this.getId()))
			.chain(acquired -> {
				if (acquired.equals(JsonValue.TRUE)) {
					return Actors.tell(this, "getSecondFork", Json.createValue(1), step);
				} else {
					if (attempt.intValue() > 5) {
						System.out.println("Warning: " + this.getId() + " has failed to acquire his first Fork " + attempt + " times");
					}
					return Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue() + 1), step);
				}
			})
			.chain(() -> {
				this.step = step;
				return Actors.State.set(this, "step", step);
			});
	}

	@Remote
	public Uni<Void> getSecondFork(JsonNumber attempt, JsonString currentStep) {
		if (!this.step.equals(currentStep)) {
			return Uni.createFrom().failure(new RuntimeException("unexpected step"));
		}
		final JsonString step = Json.createValue(UUID.randomUUID().toString());
		return Actors.call(Actors.ref("Fork", this.secondFork.getString()), "pickUp", Json.createValue(this.getId()))
			.chain(acquired -> {
				if (acquired.equals(JsonValue.TRUE)) {
					return Actors.tell(this, "eat", step);
				} else {
					if (attempt.intValue() > 5) {
						System.out.println("Warning: " + this.getId() + " has failed to acquire his second Fork " + attempt + " times");
					}
					return Actors.Reminders.schedule(this, "getSecondFork", "step", nextStepTime(), null, Json.createValue(attempt.intValue() + 1), step);
				}
			})
			.chain(() -> {
				this.step = step;
				return Actors.State.set(this, "step", step);
			});
	}

	@Remote
	public Uni<Void> eat(JsonString currentStep) {
		if (!this.step.equals(currentStep)) {
			return Uni.createFrom().failure(new RuntimeException("unexpected step"));
		}
		final JsonString step = Json.createValue(UUID.randomUUID().toString());
		if (VERBOSE) System.out.println(this.getId() + " ate serving number " + this.servingsEaten);
		return Actors.call(Actors.ref("Fork", this.secondFork.getString()), "putDown", Json.createValue(this.getId()))
			.chain(() -> Actors.call(Actors.ref("Fork", this.firstFork.getString()), "putDown", Json.createValue(this.getId())))
			.chain(() -> {
				if (this.servingsEaten.intValue() < this.targetServings.intValue()) {
					return Actors.Reminders.schedule(this, "getFirstFork", "step", nextStepTime(), null, Json.createValue(1), step);
				} else {
					return Actors.call(Actors.ref("Table", this.table.getString()), "doneEating", Json.createValue(this.getId()));
				}
			})
			.chain(() -> {
				this.servingsEaten = Json.createValue(this.servingsEaten.intValue() + 1);
				this.step = step;
				return checkpointState();
			});
	}
}
