package com.ibm.research.kar.actor;

import java.lang.invoke.MethodHandle;

public class RemoteMethod {
	private final MethodHandle method;
	private final int lockPolicy;

	public RemoteMethod(MethodHandle method, int lockPolicy) {
		this.method = method;
		this.lockPolicy = lockPolicy;
	}

	public MethodHandle getMethod() {
		return method;
	}

	public int getLockPolicy() {
		return lockPolicy;
	}
}
