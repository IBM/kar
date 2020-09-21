package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorGetState;
import static com.ibm.research.kar.Kar.actorSetState;

import javax.json.Json;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
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

	@Deactivate
	public void deactivate() {
		actorSetState(this, "inUseBy", this.inUseBy);
	}

	@Remote
	public JsonValue pickUp(JsonString who) {
		if (this.inUseBy.equals(nobody)) {
			this.inUseBy = who;
			actorSetState(this, "inUseBy", who);
			return JsonValue.TRUE;
		} else if (this.inUseBy.equals(who)) {
			return JsonValue.TRUE;
		} else {
			return JsonValue.FALSE;
		}
	}

	@Remote
	public JsonValue putDown(JsonString who) {
		if (this.inUseBy.equals(who)) {
			this.inUseBy = nobody;
			actorSetState(this, "inUseBy", this.inUseBy);
			return JsonValue.TRUE;
		} else {
			return JsonValue.FALSE;
		}
	}
}
