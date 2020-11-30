package com.ibm.research.kar;

/**
 * Configuration variables, can be overridden
 */
public class KarConfig {

	/*******
	 * settable variables in web.xml
	 */

	// default read/write connection timeout
	public static int DEFAULT_CONNECTION_TIMEOUT_MILLIS = 600000;

	// comma-delimited list of actor class names
	public static String ACTOR_CLASS_STR;

	// comma-delimited list of actor type names
	public static String ACTOR_TYPE_NAME_STR;

	// elide implementation details from actor method stack traces
	public static boolean SHORTEN_ACTOR_STACKTRACES = true;

	/********
	 * TBD settable variables microprofile-config.properties
	 */

	// maximum retries for REST Calls (only read for CDI)
	public static final int MAX_RETRY = 10;
}
