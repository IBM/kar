package test;

import static com.ibm.research.kar.Kar.call;


import javax.json.Json;
import javax.json.JsonValue;

public class RunService {

	public static void main(String[] args) {
		JsonValue params = Json.createValue(42);
		JsonValue result = (JsonValue) call("number", "number/incr", params);
		System.out.println("Got result " + result);
	}

}
