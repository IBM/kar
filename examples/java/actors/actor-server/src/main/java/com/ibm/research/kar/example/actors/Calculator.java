package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonNumber;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

/**
 * A simple calculator that performs operations on an accumulator.
 */
@Actor
public class Calculator {
	int accum;

	@Activate
	public void init() {
		accum = 0; // TODO: read from actor state
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

	@Deactivate
	public void kill() {
		// TODO: persist accum
	}
}
