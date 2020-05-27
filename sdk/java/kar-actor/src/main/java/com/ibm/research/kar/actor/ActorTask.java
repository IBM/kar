package com.ibm.research.kar.actor;

import java.lang.invoke.MethodHandle;
import java.lang.reflect.InvocationTargetException;
import java.util.concurrent.Callable;
import javax.enterprise.inject.Default;
import javax.json.JsonArray;
import com.ibm.research.kar.ActorInstance;
import com.ibm.research.kar.actor.annotations.LockPolicy;

@Default
public class ActorTask implements Callable<Object> {
	private ActorInstance actor;
	private MethodHandle actorMethod;
	private JsonArray params;
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

	public MethodHandle getActorMethod() {
		return actorMethod;
	}

	public void setActorMethod(MethodHandle actorMethod) {
		this.actorMethod = actorMethod;
	}

	public JsonArray getParams() {
		return params;
	}

	public void setParams(JsonArray params) {
		this.params = params;
	}

	@Override
	public Object call() throws Exception {
		Object[] args = new Object[params.size() + 1];
		args[0] = actor;
		for (int i = 0; i < params.size(); i++) {
			args[i + 1] = params.get(i);
		}

		try {
			Object result = null;
			if (this.lockPolicy == LockPolicy.READ) {
				result = actorMethod.invokeWithArguments(args);
			} else {
				synchronized (actor) {
					result = actorMethod.invokeWithArguments(args);
				}
			}
			return result;
		} catch (Exception e) {
			throw e;
		} catch (Throwable t) {
			throw new InvocationTargetException(t);
		}
	}
}
