package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

/**
 * A simple calculator that performs operations on an accumulator.
 */
@Actor
public class Calculator extends ActorBoilerplate {

	private int accum;

	@Activate
	public void initState() {
		JsonValue v = actorGetState(actorRef(type, id), "accum");
		if (v instanceof JsonNumber) {
			accum = ((JsonNumber)v).intValue();
		} else {
			accum = 0;
		}
	}

	@Deactivate
	public void saveState() {
		actorSetState(actorRef(type, id), "accum", Json.createValue(accum));
	}

	@Remote
	public JsonNumber add(JsonNumber num) {
		int number = num.intValue();
		accum += number;
		return Json.createValue(this.accum);
	}

	@Remote
	public JsonNumber subtract(JsonNumber num) {
		int number = num.intValue();
		accum -= number;
		return Json.createValue(accum);
	}

	@Remote
	public JsonNumber multiply(JsonNumber num) {
		int number = num.intValue();
		accum *= number;
		return Json.createValue(accum);
	}

	@Remote
	public void clear() {
		accum = 0;
	}
}
