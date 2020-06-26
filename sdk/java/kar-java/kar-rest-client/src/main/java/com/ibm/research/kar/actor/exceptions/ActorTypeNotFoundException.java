package com.ibm.research.kar.actor.exceptions;

public class ActorTypeNotFoundException extends ActorException {
	private static final long serialVersionUID = -4220811000416367515L;

	public ActorTypeNotFoundException() {
		super();
	}
	public ActorTypeNotFoundException(String errorMessage) {
        super(errorMessage);
    }
}
