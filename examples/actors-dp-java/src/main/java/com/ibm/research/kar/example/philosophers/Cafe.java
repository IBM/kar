package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorGetState;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorSetState;

import java.util.HashSet;
import java.util.Set;
import java.util.UUID;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonNumber;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

/**
 * An actor to test the performance of broadcast communication patterns
 */
@Actor
public class Cafe extends ActorSkeleton {
	Set<JsonString> diners;

	@Activate
	public void activate() {
		diners = new HashSet<JsonString>();
		JsonValue ds = actorGetState(this, "diners");
		if (ds instanceof JsonArray) {
			diners.addAll(ds.asJsonArray().getValuesAs(JsonString.class));
		}
	}

	@Deactivate
	public void deactivate() {
		checkpointState();
	}

	private void checkpointState() {
		JsonArrayBuilder jba = Json.createArrayBuilder();
		for (JsonString diner: diners) {
			jba.add(diner);
		}
		JsonArray ja = jba.build();
		actorSetState(this, "diners", ja);
	}

	@Remote
	public JsonNumber occupancy(JsonString table) {
		return Json.createValue(this.diners.size());
	}

	@Remote
	public JsonString seatTable(JsonNumber numDiners, JsonNumber servings) {
		int n = numDiners.intValue();
		int s = servings.intValue();
		System.out.println("Cafe "+this.getId()+" is seating a new table of "+n+" hungry philosophers for "+s+" servings");
		String[] philosophers = new String[n];
		String[] forks = new String[n];
		for (int i=0; i<n; i++) {
			philosophers[i] = UUID.randomUUID().toString();
			this.diners.add(Json.createValue(philosophers[i]));
			forks[i] = UUID.randomUUID().toString();
		}
		for (int i=0; i<n-1; i++) {
			actorCall(actorRef("Philosopher", philosophers[i]), "joinTable", Json.createValue(this.getId()), Json.createValue(forks[i]), Json.createValue(forks[i+1]), servings);
		}
		actorCall(actorRef("Philosopher", philosophers[n-1]), "joinTable", Json.createValue(this.getId()), Json.createValue(forks[0]), Json.createValue(forks[n-1]), servings);
		checkpointState();
		return Json.createValue("table-1");
	}

	@Remote
	public void doneEating (JsonString philosopher) {
		this.diners.remove(philosopher);
		checkpointState();
		System.out.println("Cafe "+this.getId()+": "+ philosopher.getString()+"is done eating; there are now "+diners.size()+" present");
		if (diners.size() == 0) {
			System.out.println("Cafe "+this.getId()+" is now empty!");
		}
	}
}
