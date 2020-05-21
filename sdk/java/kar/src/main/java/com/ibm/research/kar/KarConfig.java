package com.ibm.research.kar;

/**
 * Configuration variables, can be overridden
 * @author pcastro
 *
 */
public class KarConfig {
	
	/*******
	 * settable variables in web.xml
	 */
	
	// default sidecar port
	public static int DEFAULT_PORT = 3500;
	
	// default read/write connection timeout
	public static int DEFAULT_CONNECTION_TIMEOUT_MILLIS = 600000;
	
	// comma-delimited list of actor class names
	public static String ACTOR_CLASS_STR;
	
	// comma-delimited list of actor type names
	public static String ACTOR_TYPE_NAME_STR;
	
	/********
	 * TBD settable variables microprofile-config.properties
	 */

	// maximum retries for REST Calls (only read for CDI)
	public static final int MAX_RETRY = 10;
}
