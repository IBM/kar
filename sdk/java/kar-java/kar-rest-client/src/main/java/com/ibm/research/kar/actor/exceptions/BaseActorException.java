package com.ibm.research.kar.actor.exceptions;

public class BaseActorException extends Exception {
	
	public BaseActorException() {
		super();
	}

	public BaseActorException(Throwable t) {
		super(t);
	}
	
	public BaseActorException(String message) {
		super(message);
	}
}
