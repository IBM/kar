package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.util.HashMap;
import java.util.Map;

import com.ibm.research.kar.ActorInstance;

public class ActorModel {

	// KAR type
	private String type;

	// java.lang.Class for the Actor
	private Class<ActorInstance> actorClass;

	// Lookup for callable remote methods
	private Map<String, RemoteMethodType> remoteMethods;

	// Lookup for init method
	private Method activateMethod;

	// Lookup for deinit method
	private Method deactivateMethod;

	// Map of instances of this actor type indexed by id
	private Map<String, ActorInstance> actorInstances;


	public ActorModel() {
		this.remoteMethods = new HashMap<String,RemoteMethodType>();
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

	public Map<String, RemoteMethodType> getRemoteMethods() {
		return remoteMethods;
	}

	public void setRemoteMethods(Map<String, RemoteMethodType> remoteMethods) {
		this.remoteMethods = remoteMethods;
	}

	public Method getActivateMethod() {
		return activateMethod;
	}

	public void setActivateMethod(Method activateMethod) {
		this.activateMethod = activateMethod;
	}

	public Method getDeactivateMethod() {
		return deactivateMethod;
	}

	public void setDeactivateMethod(Method deactivateMethod) {
		this.deactivateMethod = deactivateMethod;
	}

	public Map<String, ActorInstance> getActorInstances() {
		return actorInstances;
	}

	public void setActorInstances(Map<String, ActorInstance> actorInstances) {
		this.actorInstances = actorInstances;
	}
}
