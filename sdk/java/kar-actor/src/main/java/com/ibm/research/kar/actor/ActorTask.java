package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.util.concurrent.Callable;

import javax.enterprise.context.RequestScoped;
import javax.enterprise.inject.Default;
import javax.json.JsonObject;

import com.ibm.research.kar.actor.annotations.LockPolicy;

@Default
public class ActorTask implements Callable<Object> {

	private Object actorObj;
	private Method actorMethod;
	private JsonObject params;
	private int lockPolicy;

	public int getLockPolicy() {
		return lockPolicy;
	}

	public void setLockPolicy(int lockPolicy) {
		this.lockPolicy = lockPolicy;
	}

	public Object getActorObj() {
		return actorObj;
	}

	public void setActorObj(Object actorObj) {
		this.actorObj = actorObj;
	}

	public Method getActorMethod() {
		return actorMethod;
	}

	public void setActorMethod(Method actorMethod) {
		this.actorMethod = actorMethod;
	}

	public JsonObject getParams() {
		return params;
	}

	public void setParams(JsonObject params) {
		this.params = params;
	}

	@Override
	public Object call() throws Exception {

		Object result = null;

		if (actorMethod.getParameterCount() > 0) {
			switch (this.lockPolicy) {
			case LockPolicy.READ:
				result = actorMethod.invoke(actorObj, params);
			case LockPolicy.WRITE:
				synchronized (actorObj) {
					result = actorMethod.invoke(actorObj, params);
				}
			}
		} else {
			switch (this.lockPolicy) {
			case LockPolicy.READ:
				result = actorMethod.invoke(actorObj);
			case LockPolicy.WRITE:
				synchronized (actorObj) {
					result = actorMethod.invoke(actorObj);
				}
			}
		}
		
		return result;


	}
}
