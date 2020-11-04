package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorRef;

import java.util.UUID;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Cafe extends ActorSkeleton {
	@Remote
	public JsonValue occupancy(JsonString table) {
		return actorCall(actorRef("Table", table.getString()), "occupancy");
	}

	@Remote
	public JsonString seatTable() {
		return seatTable(Json.createValue(5), Json.createValue(20));
	}

	@Remote
	public JsonString seatTable(JsonNumber n, JsonNumber servings) {
		JsonString requestId = Json.createValue(UUID.randomUUID().toString());
		return seatTable(n, servings, requestId);
	}

	@Remote
	public JsonString seatTable(JsonNumber n, JsonNumber servings, JsonString requestId) {
		actorCall(actorRef("Table", requestId.getString()), "prepare", Json.createValue(this.getId()), n, servings, requestId);
		return requestId;
	}
}
