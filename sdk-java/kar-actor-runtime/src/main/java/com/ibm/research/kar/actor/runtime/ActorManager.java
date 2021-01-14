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

package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;

import com.ibm.research.kar.actor.ActorInstance;

public interface ActorManager {
	// get an existing actor instance
	public ActorInstance getActor(String type, String id);

	// allocate an actor instance -- does not invoke activate
	public ActorInstance createActor(String type, String id);

	// delete an actor instance -- deos not invoke deactivate
	public boolean deleteActor(String type, String id);

	public boolean hasActorType(String type);

	public MethodHandle getActorMethod(String type, String name, int numParams);

	public MethodHandle getActorActivateMethod(String type);

	public MethodHandle getActorDeactivateMethod(String type);
}
