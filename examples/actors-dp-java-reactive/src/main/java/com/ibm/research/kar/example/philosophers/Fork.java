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

import javax.json.Json;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

import io.smallrye.mutiny.Uni;

@Actor
public class Fork extends ActorSkeleton {
	private static JsonString nobody = Json.createValue("nobody");

	private JsonString inUseBy;

	@Activate
	public Uni<Void> activate() {
		return Actors.State.get(this, "inUseBy").chain(user -> {
			if (user instanceof JsonString) {
				this.inUseBy = (JsonString)user;
			} else {
				this.inUseBy = nobody;
			}
			return Uni.createFrom().nullItem();
		});
	}

	@Remote
	public Uni<JsonValue> pickUp(JsonString who) {
		if (this.inUseBy.equals(nobody)) {
			this.inUseBy = who;
			return Actors.State.set(this, "inUseBy", who).chain(() -> Uni.createFrom().item(JsonValue.TRUE));
		} else if (this.inUseBy.equals(who)) {
			// can happen if pickUp is re-executed due to a failure
			return Uni.createFrom().item(JsonValue.TRUE);
		} else {
			return  Uni.createFrom().item(JsonValue.FALSE);
		}
	}

	@Remote
	public Uni<Void> putDown(JsonString who) {
		if (this.inUseBy.equals(who)) { // can be false if putDown is re-executed due to failure
			this.inUseBy = nobody;
			return Actors.State.setV(this, "inUseBy", this.inUseBy);
		} else {
			return Uni.createFrom().nullItem();
		}
	}
}
