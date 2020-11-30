package com.ibm.research.kar.actor.exceptions;

public class ActorMethodInvocationException extends ActorException {
	private static final long serialVersionUID = 6289655259906138150L;

	public ActorMethodInvocationException() {
		super();
	}

	public ActorMethodInvocationException(String errorMessage) {
		super(errorMessage);
	}

	public ActorMethodInvocationException(String errorMessage, Throwable cause) {
		super(errorMessage, cause);
	}
}
