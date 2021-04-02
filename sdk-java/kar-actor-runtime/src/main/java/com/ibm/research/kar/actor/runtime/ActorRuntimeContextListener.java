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

import java.util.logging.Logger;

import javax.servlet.ServletContext;
import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;

import com.ibm.research.kar.KarConfig;

/*
 * Reads configuration information from web.xml
 */
@WebListener
public class ActorRuntimeContextListener implements ServletContextListener {

	public static Logger logger = Logger.getLogger(ActorRuntimeContextListener.class.getName());

	public static final String KAR_ACTOR_CLASSES = "kar-actor-classes";
	public static final String KAR_ACTOR_TYPES = "kar-actor-types";
	public static final String KAR_SIDECAR_CONNECTION_TIMEOUT = "kar-sidecar-connection-timeout-millis";
	public static final String KAR_SHORTEN_ACTOR_STACKTRACES = "kar-shorten-actor-stacktraces";
	public static final String KAR_MAX_STACKTRACE_SIZE = "kar-max-stacktrace-size";

	@Override
	public void contextInitialized(final ServletContextEvent servletContextEvent) {
		ServletContext ctx = servletContextEvent.getServletContext();

		KarConfig.ACTOR_CLASS_STR = ctx.getInitParameter(KAR_ACTOR_CLASSES);
		KarConfig.ACTOR_TYPE_NAME_STR = ctx.getInitParameter(KAR_ACTOR_TYPES);

		String tmp = ctx.getInitParameter(KAR_SHORTEN_ACTOR_STACKTRACES);
		if (tmp != null) {
			KarConfig.SHORTEN_ACTOR_STACKTRACES = Boolean.parseBoolean(tmp);
		}

		String timeOut = ctx.getInitParameter(KAR_SIDECAR_CONNECTION_TIMEOUT);
		if (timeOut != null) {
			try {
				logger.warning("Setting sidecar connection timeout to " + timeOut);
				KarConfig.SIDECAR_CONNECTION_TIMEOUT_MILLIS = Integer.parseInt(timeOut);
			} catch (NumberFormatException ex) {
				ex.printStackTrace();
			}
		}

		String backTraceLength = ctx.getInitParameter(KAR_MAX_STACKTRACE_SIZE);
		if (backTraceLength != null) {
			try {
				logger.info("Setting max backtrace length to " + backTraceLength);
				KarConfig.MAX_STACKTRACE_SIZE = Integer.parseInt(backTraceLength);
			} catch (NumberFormatException ex) {
				ex.printStackTrace();
			}
		}

		if (System.getenv("KAR_RUNTIME_PORT") == null) {
			logger.severe("KAR_RUNTIME_PORT is not set. Fatal misconfiguration. Forcing immediate hard exit of JVM.");
			Runtime.getRuntime().halt(1);
		}
		if (System.getenv("KAR_APP_PORT") == null) {
			logger.severe("KAR_APP_PORT is not set. Fatal misconfiguration. Forcing immediate hard exit of JVM.");
			Runtime.getRuntime().halt(1);
		}
	}

}
