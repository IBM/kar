package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.util.Map;

public class ActorReference {
	private String type;
	private String id;
	private Class<?> cls;
	private Object actorInstance;
	private String className;
	private Map<String, Method> remoteMethods;
	private Method activateMethod;
	private Method deactivateMethod;
	
	
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

	public Map<String,Method> getRemoteMethods() {
		return remoteMethods;
	}
	
	public Method getMethod(String name) {
		return remoteMethods.get(name);
	}
	
	public void setRemoteMethods(Map<String, Method> remoteMethods) {
		this.remoteMethods = remoteMethods;
	}
	public String getClassName() {
		return className;
	}
	public void setClassName(String className) {
		this.className = className;
	}
	public Object getActorInstance() {
		return actorInstance;
	}
	public void setActorInstance(Object actorInstance) {
		this.actorInstance = actorInstance;
	}
	public String getType() {
		return type;
	}
	public void setType(String type) {
		this.type = type;
	}
	public String getId() {
		return id;
	}
	public void setId(String id) {
		this.id = id;
	}
	public Class<?> getCls() {
		return cls;
	}
	public void setCls(Class<?> cls) {
		this.cls = cls;
	}

}