package com.ibm.research.kar.actor;

import java.lang.reflect.Method;
import java.util.concurrent.Callable;

import javax.enterprise.context.RequestScoped;
import javax.enterprise.inject.Default;
import javax.json.JsonObject;

@Default
public class ActorTask implements Callable<Object> {

	private Object actorObj;
	private Method actorMethod;
	private JsonObject params;

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

		if (actorMethod.getParameterCount() > 0) {
			synchronized (actorObj) {
				return actorMethod.invoke(actorObj, params);
			}
		} else {
			synchronized (actorObj) {
				return actorMethod.invoke(actorObj);
			}
		}
	}


}
