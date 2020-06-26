package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorTell;

import java.util.Random;
import java.util.concurrent.atomic.AtomicLong;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;
import com.ibm.research.kar.actor.exceptions.ActorException;

@Actor
public class BroadcastTest extends ActorBoilerplate {

	@Activate
	public void init() {
	}


	// Broadcast Spread -------------------------------

	AtomicLong driver_expecting;
	Object lock = new Object();

	@Remote
	public JsonValue spread(JsonObject json) {
		int ndrivers = json.getInt("ndrivers");
		int nleaves = json.getInt("nleaves");
		int population = json.getInt("population");
		String syncOrAsync = json.getString("syncOrAsync", "sync");
		boolean trace = json.getBoolean("trace", false);
		String session = this.getSession();

		int maxParallel;
		if (syncOrAsync.equals("sync")) {
			maxParallel = ndrivers;
		}
		else {
			maxParallel = ndrivers * nleaves;
		}
		driver_expecting = new AtomicLong(maxParallel);

		System.out.println("Broadcast type:" + syncOrAsync + " #drivers:" + ndrivers + " #leaves:" + nleaves
				+ " population:" + population + " max parallel:" + maxParallel);

		long starttime = System.nanoTime();

		JsonObject params = Json.createObjectBuilder()
				.add("nleaves", nleaves)
				.add("population", population)
				.add("syncOrAsync", syncOrAsync)
				.add("trace", trace)
				.add("session",  session)
				.build();

		for (int driver=0; driver<ndrivers; driver++) {
			actorTell(actorRef("broadcast", "D"+driver), "driver", params);
		}

		synchronized (lock) {
			try {
				lock.wait(30000);
			} catch (InterruptedException e) {	}
		}

		long duration = 0;
		duration = (System.nanoTime()-starttime)/1000000;
		long left=driver_expecting.get();
		if (left == 0) {
			System.out.println("broadcast took " + duration + "ms");
			JsonObject result = Json.createObjectBuilder()
					.add("syncOrAsync", syncOrAsync)
					.add("maxParallel", maxParallel)
					.add("duration", duration)
					.build();
			return result;
		}
		else {
			System.out.println("broadcast timed out at " + duration + "ms with "+left+" replies outstanding");
			JsonObject result = Json.createObjectBuilder()
					.add("recursion timed out at", duration)
					.build();
			return result;
		}
	}

	@Remote
	public void leafdone() {
		synchronized(lock) {
			if (driver_expecting.decrementAndGet() <= 0) {
				lock.notify();
			}
		}
	}


	// Broadcast Driver -------------------------------

	@Remote
	public JsonValue driver(JsonObject json) {
		int nleaves = json.getInt("nleaves");
		int population = json.getInt("population");
		String syncOrAsync = json.getString("syncOrAsync", "sync");
		boolean trace = json.getBoolean("trace", false);
		String session = json.getString("session");

		long starttime = System.nanoTime();

		if (trace) {
			System.out.println("driver " + this.getId() + " starting");
		}

		JsonObject params = Json.createObjectBuilder()
				.add("session",  session)
				.build();

		Random random = new Random();
		for (int i=0; i<nleaves; i++) {
			int target = random.nextInt(population);
			try {
				if (syncOrAsync.equals("sync")) {
					actorCall(actorRef("broadcast", "SL"+target), "syncleaf");
				}
				else {
					actorTell(actorRef("broadcast", "SL"+target), "asyncleaf", params);
				}
			} catch (ActorException e) {
				System.err.println("Broadcast sync: error calling " + "actorRef(\"broadcast\",SL"+target+" Error: " +e.toString());
				params = Json.createObjectBuilder()
						.add("error", e.toString())
						.build();
				return params;
			}
		}

		if (trace) {
			System.out.println("driver " + this.getId() + " finished ");
		}

		if (syncOrAsync.equals("sync")) {
			try {
				actorCall(session, actorRef("broadcast","1"), "leafdone");
			} catch (ActorException e) {
				System.err.println(e.toString());
				JsonObject result = Json.createObjectBuilder()
						.add("Error", e.toString())
						.build();
				return result;
			}
		}

		long duration = 0;
		duration = (System.nanoTime()-starttime)/1000000;

		if (trace) {
			System.out.println("driver " + this.getId() + " took " + duration + "ms");
		}

		JsonObject result = Json.createObjectBuilder()
				.add("driver " + this.getId() + " duration", duration)
				.build();
		return result;
	}


	@Remote
	public void syncleaf() {
		// nada
	}


	@Remote
	public void asyncleaf(JsonObject json) {
		String session = json.getString("session");
		try {
			actorCall(session, actorRef("broadcast","1"), "leafdone");
		} catch (ActorException e) {
			System.err.println(e.toString());
			return;
		}
	}


	@Deactivate
	public void kill() {
	}
}
