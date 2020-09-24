package com.ibm.research.kar.standalone.test;

import static com.ibm.research.kar.standalone.Kar.call;
import static com.ibm.research.kar.standalone.Kar.init;

import javax.json.Json;
import javax.json.JsonValue;

public class RunService {

	public static void main(String[] args) {
		init();
		JsonValue params = Json.createValue(8);
		JsonValue result = (JsonValue) call("number", "number/incr", params);
		System.out.println("Got result " + result);
	}

}
