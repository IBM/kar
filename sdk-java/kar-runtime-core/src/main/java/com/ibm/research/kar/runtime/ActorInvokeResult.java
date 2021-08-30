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

/**
 * An object representing the result of invoking an Actor method.
 * If `error` is false, then `value` contains the result of the method
 */
public class ActorInvokeResult {
  public final boolean error;
  public final Object value;
  public final String message;
  public final String stack;

  ActorInvokeResult(Object value) {
    this.error = false;
    this.value = value;
    this.message = null;
    this.stack = null;
  }

  ActorInvokeResult(String message, String stack) {
    this.error = true;
    this.value = false;
    this.message = message;
    this.stack = stack;
  }

}
