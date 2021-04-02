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

package com.ibm.research.kar;

/**
 * Configuration variables, can be overridden
 */
public class KarConfig {

	/*******
	 * settable variables in web.xml
	 */

	// comma-delimited list of actor class names
	public static String ACTOR_CLASS_STR;

	// comma-delimited list of actor type names
	public static String ACTOR_TYPE_NAME_STR;

	// default read/write connection timeout. 0 means infinite
	public static int SIDECAR_CONNECTION_TIMEOUT_MILLIS = 0;

	// elide implementation details from actor method stack traces
	public static boolean SHORTEN_ACTOR_STACKTRACES = true;

	// truncate really long stack traces to avoid exceeding maximum Kafka message size.
	public static int MAX_STACKTRACE_SIZE = 500 * 1024;

	/********
	 * TBD settable variables microprofile-config.properties
	 */

	// maximum retries for REST Calls (only read for CDI)
	public static final int MAX_RETRY = 10;
}
