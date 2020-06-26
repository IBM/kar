package test;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorRef;

import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorCall;

public class RunActor {

	public static void main(String[] args) {
		JsonObject params = Json.createObjectBuilder().add("number", 42).build();

		ActorRef dummy = actorRef("dummy", "dummyid");
		JsonValue result = actorCall(dummy, "canBeInvoked", params);
		System.out.println(result);
	}

}
