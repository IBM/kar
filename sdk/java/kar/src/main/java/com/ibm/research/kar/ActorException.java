package com.ibm.research.kar;

public class ActorException extends Exception {

	public ActorException(Throwable t) {
		super(t);
	}
	
	public ActorException(String message) {
		super(message);
	}
}
