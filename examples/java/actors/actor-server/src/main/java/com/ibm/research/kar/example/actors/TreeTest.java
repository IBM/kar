package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorTell;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.ActorMethodNotFoundException;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class TreeTest extends ActorBoilerplate {

	@Activate
	public void init() {
	}

	// Sync actor tree -------------------------------

	@Remote
	public JsonValue testsync(JsonObject json) {
		int depth = json.getInt("depth");

		int expecting = 1 << (depth-1);
		System.out.println("Sync test expecting " + expecting + " leaves");
		long starttime = System.nanoTime();

		JsonObject params = Json.createObjectBuilder()
				.add("depth", depth)
				.build();

		try {
			actorCall(this, actorRef("treetest", "1"), "forksync", params);
		} catch (ActorMethodNotFoundException e) {
			params = Json.createObjectBuilder()
					.add("error", e.toString())
					.build();
			return params;
		}

		long duration = 0;
		duration = (System.nanoTime()-starttime)/1000000;
		System.out.println("recursion took " + duration + "ms");

		JsonObject result = Json.createObjectBuilder()
				.add("depth", depth)
				.add("Sync duration", duration)
				.build();
		return result;
	}

	@Remote
	public void forksync(JsonObject json) {
		int depth = json.getInt("depth");

		if (--depth > 0) {
			JsonObject params = Json.createObjectBuilder()
					.add("depth", depth)
					.build();

			try {
				actorCall(actorRef("treetest", Integer.toString((Integer.valueOf(this.getId()) * 2))), "forksync", params);
				actorCall(actorRef("treetest", Integer.toString((Integer.valueOf(this.getId()) * 2 + 1))), "forksync", params);
			} catch (Exception e) {
				System.err.println(e.toString());
				return;
			}
		}
	}


	// Async tree ----------------------------------

	int async_expecting;
	Object lock = new Object();
	@Remote
	public JsonValue testasync(JsonObject json) {
		int depth = json.getInt("depth");
		String session = this.getSession();

		async_expecting = 1 << (depth-1);
		if (! this.getId().equals("1")) {
			System.err.println("testasync called with ID:"+this.getId()+" instead of:1");
			JsonObject result = Json.createObjectBuilder()
					.add("ERROR", "testasync must be called with ID=\"1\"")
					.build();
			return result;
		}
		System.out.println("Async test expecting " + async_expecting + " leaves");
		long starttime = System.nanoTime();

		JsonObject params = Json.createObjectBuilder()
				.add("depth", depth)
				.add("session",  session)
				.build();

		try {
			actorCall(this, actorRef("treetest", "1"), "forkasync", params);
		} catch (ActorMethodNotFoundException e1) {
			params = Json.createObjectBuilder()
					.add("error", e1.toString())
					.build();
			return params;
		}

		synchronized (lock) {
			try {
				lock.wait(30000);
			} catch (InterruptedException e) {	}
		}

		long duration = 0;
		duration = (System.nanoTime()-starttime)/1000000;
		int left=async_expecting;
		if (left == 0) {
			System.out.println("recursion took " + duration + "ms");
			JsonObject result = Json.createObjectBuilder()
					.add("depth", depth)
					.add("Async duration", duration)
					.build();
			return result;
		}
		else {
			System.out.println("recursion timed out at " + duration + "ms with "+left+" leaves remaining");
			JsonObject result = Json.createObjectBuilder()
					.add("recursion timed out at", duration)
					.build();
			return result;
		}
	}

	@Remote
	public void leafdone() {
		if (--async_expecting <= 0) {
			synchronized(lock) {
				lock.notify();
			}
		}
	}

	@Remote
	public void forkasync(JsonObject json) {
		int depth = json.getInt("depth");
		String session = json.getString("session");

		if (--depth == 0) {
			try {
				actorCall(session, actorRef("treetest","1"), "leafdone");
			} catch (ActorMethodNotFoundException e) {
				System.err.println(e.toString());
				return;
			}
		}
		else {
			JsonObject params = Json.createObjectBuilder()
					.add("depth", depth)
					.add("session",  session)
					.build();

			actorTell(actorRef("treetest", Integer.toString((Integer.valueOf(this.getId()) * 2))), "forkasync", params);
			actorTell(actorRef("treetest", Integer.toString((Integer.valueOf(this.getId()) * 2 + 1))), "forkasync", params);
		}
		return;
	}


	@Deactivate
	public void kill() {
	}
}
