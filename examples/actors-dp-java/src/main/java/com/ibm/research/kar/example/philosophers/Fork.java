package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorGetState;
import static com.ibm.research.kar.Kar.actorSetState;

import javax.json.Json;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Fork extends ActorSkeleton {
	private static JsonString nobody = Json.createValue("nobody");

	private JsonString inUseBy;

	@Activate
	public void activate() {
		JsonValue user = actorGetState(this, "inUseBy");
		if (user instanceof JsonString) {
			this.inUseBy = (JsonString)user;
		} else {
			this.inUseBy = nobody;
		}
	}

	@Remote
	public JsonValue pickUp(JsonString who) {
		if (this.inUseBy.equals(nobody)) {
			this.inUseBy = who;
			actorSetState(this, "inUseBy", who);
			return JsonValue.TRUE;
		} else if (this.inUseBy.equals(who)) {
			// can happen if pickUp is re-executed due to a failure
			return JsonValue.TRUE;
		} else {
			return JsonValue.FALSE;
		}
	}

	@Remote
	public void putDown(JsonString who) {
		if (this.inUseBy.equals(who)) { // can be false if putDown is re-executed due to failure
			this.inUseBy = nobody;
			actorSetState(this, "inUseBy", this.inUseBy);
		}
	}
}
