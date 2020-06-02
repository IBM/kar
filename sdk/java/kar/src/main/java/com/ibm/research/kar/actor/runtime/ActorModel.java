package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;
import java.util.HashMap;
import java.util.Map;

import com.ibm.research.kar.actor.ActorInstance;

public class ActorModel {

	// KAR type
	private String type;

	// java.lang.Class for the Actor
	private Class<ActorInstance> actorClass;

	// Lookup for callable remote methods
	private Map<String, MethodHandle> remoteMethods;

	// Lookup for init method
	private MethodHandle activateMethod;

	// Lookup for deinit method
	private MethodHandle deactivateMethod;

	// Map of instances of this actor type indexed by id
	private Map<String, ActorInstance> actorInstances;


	public ActorModel() {
		this.remoteMethods = new HashMap<String,MethodHandle>();
		this.actorInstances = new HashMap<String,ActorInstance>();
	}

	/*
	 * Getters and Setters
	 */

	public String getType() {
		return type;
	}

	public void setType(String type) {
		this.type = type;
	}


	public Class<ActorInstance> getActorClass() {
		return actorClass;
	}

	public void setActorClass(Class<ActorInstance> cls) {
		this.actorClass = cls;
	}

	public Map<String, MethodHandle> getRemoteMethods() {
		return remoteMethods;
	}

	public void setRemoteMethods(Map<String, MethodHandle> remoteMethods) {
		this.remoteMethods = remoteMethods;
	}

	public MethodHandle getActivateMethod() {
		return activateMethod;
	}

	public void setActivateMethod(MethodHandle activateMethod) {
		this.activateMethod = activateMethod;
	}

	public MethodHandle getDeactivateMethod() {
		return deactivateMethod;
	}

	public void setDeactivateMethod(MethodHandle deactivateMethod) {
		this.deactivateMethod = deactivateMethod;
	}

	public Map<String, ActorInstance> getActorInstances() {
		return actorInstances;
	}

	public void setActorInstances(Map<String, ActorInstance> actorInstances) {
		this.actorInstances = actorInstances;
	}
}
