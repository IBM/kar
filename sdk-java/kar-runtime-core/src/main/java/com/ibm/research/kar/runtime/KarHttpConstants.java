/*
 * Copyright IBM Corporation 2020,2023
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
 * Constants used to encode HTTP responses.
 */
public interface KarHttpConstants {
	public static final String KAR_ACTOR_JSON = "application/kar+json";
  public static final String TEXT_PLAIN = "text/plain";

  public static final int OK = 200;
  public static final int CREATED = 201;
  public static final int ACCEPTED = 202;
  public static final int NO_CONTENT = 204;

  public static final int BAD_REQUEST = 400;
  public static final int NOT_FOUND = 404;
  public static final int REQUEST_TIMEOUT = 408;
}
