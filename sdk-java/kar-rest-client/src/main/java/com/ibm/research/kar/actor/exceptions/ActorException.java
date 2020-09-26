package com.ibm.research.kar.actor.exceptions;

public class ActorException extends RuntimeException {
	private static final long serialVersionUID = 5573935028054003171L;

	public ActorException() {
		super();
	}

	public ActorException(Throwable t) {
		super(t);
	}

	public ActorException(String message) {
		super(message);
	}

	public ActorException(String message, Throwable cause) {
		super(message, cause);
	}
}
