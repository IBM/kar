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

import java.time.Duration;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
import com.ibm.research.kar.Kar.Actors.TellContinueResult;
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
		JsonObject state = jb.build();
		return Actors.State.setV(this, state);
	}

	private Duration thinkTime() {
		return Duration.ofMillis(1 + (long)(Math.random() * 1000)); // random 1...1000ms
	}

	@Remote
	public Uni<TellContinueResult> joinTable(JsonString table, JsonString firstFork, JsonString secondFork, JsonNumber targetServings) {
		this.table = table;
		this.firstFork = firstFork;
		this.secondFork = secondFork;
		this.servingsEaten = Json.createValue(0);
		this.targetServings = targetServings;
		return checkpointState()
			.onItem().delayIt().by(thinkTime())
			.chain(() -> Actors.continuation(this, "getFirstFork", Json.createValue(1)));
	}

	@Remote
	public Uni<TellContinueResult> getFirstFork(JsonNumber attempt) {
		return Actors.call(Actors.ref("Fork", this.firstFork.getString()), "pickUp", Json.createValue(this.getId()))
			.chain(acquired -> {
				if (acquired.equals(JsonValue.TRUE)) {
					return Actors.continuation(this, "getSecondFork", Json.createValue(1));
				} else {
					if (attempt.intValue() > 5) {
						System.out.println("Warning: " + this.getId() + " has failed to acquire his first Fork " + attempt + " times");
					}
					return Uni.createFrom().nullItem().onItem().delayIt().by(thinkTime())
						.chain(() -> Actors.continuation(this, "getFirstFork", Json.createValue(attempt.intValue() + 1)));
				}
			});
	}

	@Remote
	public Uni<TellContinueResult> getSecondFork(JsonNumber attempt) {
		return Actors.call(Actors.ref("Fork", this.secondFork.getString()), "pickUp", Json.createValue(this.getId()))
			.chain(acquired -> {
				if (acquired.equals(JsonValue.TRUE)) {
					return Actors.continuation(this, "eat", this.servingsEaten);
				} else {
					if (attempt.intValue() > 5) {
						System.out.println("Warning: " + this.getId() + " has failed to acquire his second Fork " + attempt + " times");
					}
					return Uni.createFrom().nullItem().onItem().delayIt().by(thinkTime())
						.chain(() -> Actors.continuation(this, "getSecondFork", Json.createValue(attempt.intValue() + 1)));
				}
			});
	}

	@Remote
	public Uni<TellContinueResult> eat(JsonNumber serving) {
		if (!serving.equals(this.servingsEaten)) return null; // squash re-execution (must have failed after State.set below, but before TCR was committed)
		if (VERBOSE) System.out.println(this.getId() + " ate serving number " + this.servingsEaten);
		return Actors.call(Actors.ref("Fork", this.secondFork.getString()), "putDown", Json.createValue(this.getId()))
			.chain(() -> Actors.call(Actors.ref("Fork", this.firstFork.getString()), "putDown", Json.createValue(this.getId())))
			.chain(() -> {
				this.servingsEaten = Json.createValue(serving.intValue() + 1);
				return Actors.State.set(this, "servingsEaten", this.servingsEaten);
			}).chain(() -> {
				if (this.servingsEaten.intValue() < this.targetServings.intValue()) {
					return Uni.createFrom().nullItem().onItem().delayIt().by(thinkTime())
						.chain(() -> Actors.continuation(this, "getFirstFork", Json.createValue(1)));
				} else {
					return Actors.continuation(Actors.ref("Table", this.table.getString()), "doneEating", Json.createValue(this.getId()));
				}
			});
	}
}
