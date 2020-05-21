package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.util.concurrent.Callable;

import javax.enterprise.inject.Default;
import javax.json.JsonValue;

import com.ibm.research.kar.ActorInstance;
import com.ibm.research.kar.actor.annotations.LockPolicy;

@Default
public class ActorTask implements Callable<Object> {

	private ActorInstance actor;
	private Method actorMethod;
	private JsonValue params;
	private int lockPolicy;

	public int getLockPolicy() {
		return lockPolicy;
	}

	public void setLockPolicy(int lockPolicy) {
		this.lockPolicy = lockPolicy;
	}

	public ActorInstance getActor() {
		return actor;
	}

	public void setActor(ActorInstance actorObj) {
		this.actor = actorObj;
	}

	public Method getActorMethod() {
		return actorMethod;
	}

	public void setActorMethod(Method actorMethod) {
		this.actorMethod = actorMethod;
	}

	public JsonValue getParams() {
		return params;
	}

	public void setParams(JsonValue params) {
		this.params = params;
	}

	@Override
	public Object call() throws Exception {

		Object result = null;

		if (actorMethod.getParameterCount() > 0) {
			switch (this.lockPolicy) {
			case LockPolicy.READ:
				result = actorMethod.invoke(actor, params);
				break;
			default:
				synchronized (actor) {
					result = actorMethod.invoke(actor, params);
			}
		}
		} else {
			switch (this.lockPolicy) {
			case LockPolicy.READ:
				result = actorMethod.invoke(actor);
				break;
			default:
				synchronized (actor) {
					result = actorMethod.invoke(actor);
			}
		}
		}

		return result;
	}
}
