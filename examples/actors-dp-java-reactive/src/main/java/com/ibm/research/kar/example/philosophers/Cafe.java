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

import java.util.UUID;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

import io.smallrye.mutiny.Uni;

@Actor
public class Cafe extends ActorSkeleton {
	@Remote
	public Uni<JsonValue> occupancy(JsonString table) {
		return Actors.call(Actors.ref("Table", table.getString()), "occupancy");
	}

	@Remote
	public Uni<JsonString> seatTable() {
		return seatTable(Json.createValue(5), Json.createValue(20));
	}

	@Remote
	public Uni<JsonString> seatTable(JsonNumber n, JsonNumber servings) {
		JsonString requestId = Json.createValue(UUID.randomUUID().toString());
		return seatTable(n, servings, requestId);
	}

	@Remote
	public Uni<JsonString> seatTable(JsonNumber n, JsonNumber servings, JsonString requestId) {
		return Actors.call(Actors.ref("Table", requestId.getString()), "prepare", Json.createValue(this.getId()), n, servings)
			.chain(() -> Uni.createFrom().item(requestId));
	}
}
