package com.ibm.research.kar.actor.exceptions;

public class ActorMethodTimeoutException extends ActorException {
  private static final long serialVersionUID = 6500632661735748511L;

  public ActorMethodTimeoutException() {
		super();
	}

	public ActorMethodTimeoutException(String errorMessage) {
		super(errorMessage);
	}
}
