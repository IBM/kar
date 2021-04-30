package com.ibm.research.kar;

import io.smallrye.mutiny.Uni;
import io.vertx.mutiny.core.MultiMap;
import io.vertx.mutiny.core.buffer.Buffer;

public class Kar {

	private final static String KAR_API_CONTEXT_ROOT = "/kar/v1";

    private static KarRest karClient = KarRest.getClient();

    /******************
	 * KAR API
	 ******************/

	/**
	 * KAR API methods for Services
	 */
	public static class Services {
		/*
		 * Lower-level REST operations on a KAR Service
		 */

		/**
		 * Synchronous REST DELETE
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Buffer delete(String service, String path) {
			path = Kar.getServicePath(service, path);
			Uni<Buffer> uni = karClient.callDelete(path, getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST DELETE
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<Buffer> deleteAsync(String service, String path) {
			path = Kar.getServicePath(service, path);
			return karClient.callDelete(path, getStandardHeaders());
		}
		
		/**
		 * Synchronous REST GET
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Object get(String service, String path) {
			path = Kar.getServicePath(service, path);
			Uni<Object> uni = karClient.callGet(path, null, Kar.getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST GET
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<Object> getAsync(String service, String path) {
			path = Kar.getServicePath(service, path);
			return karClient.callGet(path, null, Kar.getStandardHeaders());
		}

		/**
		 * Synchronous REST HEAD
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Object head(String service, String path) {
			path = Kar.getServicePath(service, path);
			Uni<Object> uni = karClient.callHead(path, getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST HEAD
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<Object> headAsync(String service, String path) {
			path = Kar.getServicePath(service, path);
			return karClient.callHead(path, getStandardHeaders());
		}

		/**
		 * Synchronous REST PATCH
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Object patch(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			Uni<Object> uni = karClient.callPatch(path, body, getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST PATCH
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<Object> patchAsync(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			return karClient.callPatch(path, body, getStandardHeaders());
		}

		/**
		 * Synchronous REST POST
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Object post(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			Uni<Object> uni = karClient.callPost(path, body, Kar.getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST POST
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<Object> postAsync(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			return karClient.callPost(path, body, Kar.getStandardHeaders());
		}

		/**
		 * Synchronous REST PUT
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Object put(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			Uni<Object> uni = karClient.callPut(path, body, getStandardHeaders());
			return uni.subscribeAsCompletionStage().join();
		}

		/**
		 * Asynchronous REST PUT
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<Object> putAsync(String service, String path, Object body) {
			path = Kar.getServicePath(service, path);
			return karClient.callPut(path, body, getStandardHeaders());
		}
	}

	/**
	 * KAR API methods for Actors
	 */
	public static class Actors {
	}


	/*
	  Utility calls
	*/

	/**
	 * Construct sidecar URI from service name and path for service calls
	 * @param service
	 * @param path
	 * @return path component of sidecar REST call
	 */
	private static String getServicePath(String service, String path) {
        return Kar.KAR_API_CONTEXT_ROOT + "/service/" + service + "/call/" + path;
    }

	/**
	 * Return headers used by KAR calls for service calls
	 * @return headers
	 */
    private static MultiMap getStandardHeaders() {
        MultiMap headers = MultiMap.caseInsensitiveMultiMap();
        headers.add("Content-type", "application/json; charset=utf-8");
        //headers.add("Accept", "application/json; charset=utf-8");
        return headers;
    }
    
}
