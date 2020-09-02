package com.ibm.research.kar.actor.exceptions;

public class ActorMethodNotFoundException extends ActorException {
	private static final long serialVersionUID = -4620782613715900058L;

	public ActorMethodNotFoundException() {
		super();
	}

	public ActorMethodNotFoundException(String errorMessage) {
		super(errorMessage);
	}
}
