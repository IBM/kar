package com.ibm.research.kar.actor;

import java.lang.reflect.Method;

public class RemoteMethodType {
	
	private Method method;
	private int lockPolicy;
	
	public Method getMethod() {
		return method;
	}
	public void setMethod(Method method) {
		this.method = method;
	}
	public int getLockPolicy() {
		return lockPolicy;
	}
	public void setLockPolicy(int lockPolicy) {
		this.lockPolicy = lockPolicy;
	}
	
	

}
