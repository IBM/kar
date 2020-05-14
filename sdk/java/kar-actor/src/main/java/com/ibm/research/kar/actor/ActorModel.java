package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class ActorModel {
	
	// KAR type
	private String type;
	
	// Class name and references
	private String className;

	// Lookup for callable remote methods
	private Map<String, RemoteMethodType> remoteMethods;
	
	// Lookup for init method
	private Method activateMethod;
	
	// Lookup for deinit method
	private Method deactivateMethod;
	
	private Type[] interfaces;
	

	public Type[] getInterfaces() {
		return interfaces;
	}
	public void setInterfaces(Type[] interfaces) {
		this.interfaces = interfaces;
	}


	// Map of instances of this actor type indexed by id
	private Map<String, Object> actorInstances;


	public ActorModel() {
		this.remoteMethods = new HashMap<String,RemoteMethodType>();
		this.actorInstances = new HashMap<String,Object>();
	}
	/**
	 * Getters and Setters
	 * 
	 */
	 
	public String getType() {
		return type;
	}


	public void setType(String type) {
		this.type = type;
	}


	public String getClassName() {
		return className;
	}


	public void setClassName(String className) {
		this.className = className;
	}


	public Class<?> getActorClass() {
		try {
			return Class.forName(this.className);
		} catch (ClassNotFoundException e) {
			e.printStackTrace();
		}
		
		return null;
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


	public Map<String, Object> getActorInstances() {
		return actorInstances;
	}


	public void setActorInstances(Map<String, Object> actorInstances) {
		this.actorInstances = actorInstances;
	}
	
	

}
