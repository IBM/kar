package com.ibm.research.kar.example.actors;

import static com.ibm.research.kar.Kar.actorGetAllState;
import static com.ibm.research.kar.Kar.actorGetState;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorSetMultipleState;
import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorTell;

import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.atomic.AtomicLong;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.ActorRef;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.LockPolicy;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class TreeActorParallel extends ActorBoilerplate {

	AtomicLong expectedLeaves;
	Boolean waiter;
	
	@Activate
	public void init() {
	}

	private void propagate(String label, int level, int maxdepth, String topid, String session, boolean trace) {
		ActorRef actorA = actorRef("telltree",label+level+"A");
		ActorRef actorB = actorRef("telltree",label+level+"B");	

		JsonObject paramsA = Json.createObjectBuilder()
				.add("label",  label+level+"A")
				.add("level", level+1)
				.add("maxdepth", maxdepth)
				.add("topid", topid)
				.add("session",  session)
				.add("trace", trace)
				.build();
		actorTell(actorA, "tellTree", paramsA);

		JsonObject paramsB = Json.createObjectBuilder()
				.add("label",  label+level+"B")
				.add("level", level+1)
				.add("maxdepth", maxdepth)
				.add("topid", topid)
				.add("session",  session)
				.add("trace", trace)
				.build();
		actorTell(actorB, "tellTree", paramsB);
	}
	
	@Remote
	public void tellDone(JsonObject json) {
		String leaf = json.getString("leaf");
		boolean trace = json.getBoolean("trace", false);

		long expecting = expectedLeaves.decrementAndGet();
		if (trace)
			System.out.println("TreeActor:tellDone notified by " + leaf + " still expecting " + expecting);
	}
	
	@Remote
	public JsonValue callTree(JsonObject json) {
		int maxdepth = json.getInt("maxdepth");
		boolean trace = json.getBoolean("trace", false);

		String label = "top";
		int level = 1;
		String topid = this.getId();
		String session = this.getSession();
		long starttime = System.nanoTime();
		expectedLeaves = new AtomicLong();
		expectedLeaves.set(1 << (maxdepth-1)); 
		waiter = new Boolean(true);

		System.out.println("TreeActor:callTree " + this.getId() + " Expecting " + expectedLeaves.get() + " notifies");
		JsonObject params = Json.createObjectBuilder()
				.add("expected", this.expectedLeaves.get())
				.add("trace", trace)
				.build();
		ActorRef countActor = actorRef("treecounter", "tellTreeCounter");
		actorCall(countActor, "setCount", params);
		

		propagate(label, level, maxdepth, topid, session, trace);

		int timeout = 300;
		while (expectedLeaves.get() > 0) {
			try {
				Thread.sleep(100);
				if (0 >= --timeout) {
					JsonObject result = Json.createObjectBuilder()
							.add("duration", "timed out")
							.build();
					return result;
				}
			} catch (InterruptedException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			}
			if (0 == timeout%10 )
				System.out.println("TreeActor:callTree Still expecting " + expectedLeaves + " notifies");
		}
		
		long duration = (System.nanoTime()-starttime)/1000000;
		System.out.println("recursion took " + duration + "ms");

		JsonObject result = Json.createObjectBuilder()
				.add("duration", duration)
				.build();
		return result;
	}

	@Remote
	public void tellTree(JsonObject json) {
		String label = json.getString("label");
		int level = json.getInt("level");
		int maxdepth = json.getInt("maxdepth");
		String topid = json.getString("topid");
		String session = json.getString("session");
		boolean trace = json.getBoolean("trace", false);

		if (trace)
			System.out.println("TreeActor:tellTree " + this.getId() + " Received level " + level);

		if (level >= maxdepth) {
			JsonObject params = Json.createObjectBuilder()
					.add("leaf", this.getId())
					.add("trace", trace)
					.build();

			ActorRef countActor = actorRef("treecounter", "tellTreeCounter");
			actorCall(countActor, "callDone", params);
			ActorRef topActor = actorRef("telltree", topid);
			actorCall(session, topActor, "tellDone", params);
			return;
		}

		propagate(label, level, maxdepth, topid, session, trace);
		return;
	}

	public void cannotBeInvoked() {
	}

	@Remote
	public JsonNumber setState(JsonObject updates) {
		int numNew = actorSetMultipleState(this, updates);
		return Json.createValue(numNew);
	}

	@Remote
	public JsonValue getStateElement(JsonString key) {
		return actorGetState(this, key.getString());
	}

	@Remote
	public JsonNumber printState() {
		Map<String,JsonValue> state = actorGetAllState(this);
		for (Entry<String,JsonValue> e: state.entrySet()) {
			System.out.println(e.getKey() + " = " + e.getValue().toString());
		}
		return Json.createValue(state.size());
	}

	@Deactivate
	public void kill() {
	}
}
