/*
 * Copyright IBM Corporation 2020,2021
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.ibm.research.kar.runtime;

import java.lang.invoke.MethodHandle;
import java.util.Map;

import com.ibm.research.kar.actor.ActorInstance;

/**
 * An ActorType instance contains the Class and MethodHandle objects
 * that are used to create actor instances and invoke actor methods.
 */
public final class ActorType {

	// KAR type
	private final String type;

	// java.lang.Class for the Actor
	private final Class<ActorInstance> actorClass;

	// Lookup for callable remote methods
	private final Map<String, MethodHandle> remoteMethods;

	// Lookup for init method
	private final MethodHandle activateMethod;

	// Lookup for deinit method
	private final MethodHandle deactivateMethod;

	public ActorType(String type, Class<ActorInstance> cls, Map<String, MethodHandle> methods, MethodHandle activate, MethodHandle deactivate) {
		this.type = type;
		this.actorClass = cls;
		this.remoteMethods = methods;
		this.activateMethod = activate;
		this.deactivateMethod = deactivate;
	}

	public String getType() {
		return type;
	}

	public Class<ActorInstance> getActorClass() {
		return actorClass;
	}

	public Map<String, MethodHandle> getRemoteMethods() {
		return remoteMethods;
	}

	public MethodHandle getActivateMethod() {
		return activateMethod;
	}

	public MethodHandle getDeactivateMethod() {
		return deactivateMethod;
	}
}
