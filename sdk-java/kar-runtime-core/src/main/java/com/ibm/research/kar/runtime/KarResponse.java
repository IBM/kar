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
 * A framework agnostic representation of an HttpResponse.
 */
public class KarResponse {
	public static final String KAR_ACTOR_JSON = "application/kar+json";
  public static final String TEXT_PLAIN = "text/plain";

  public static final int OK = 200;
  public static final int CREATED = 201;
  public static final int ACCEPTED = 202;
  public static final int NO_CONTENT = 204;

  public static final int BAD_REQUEST = 400;
  public static final int NOT_FOUND = 404;
  public static final int REQUEST_TIMEOUT = 408;

  public static final int INTERNAL_ERROR = 500;

  /**
   * The http status code for this response
   */
  public final int statusCode;

  /**
   * The body of the response.
   *
   * If the response is to an HTTP operation that does not expect a body, this may null.
   * If the response is to an HTTP operation that does expect a body, a value of
   * `null` is interpreted as Json.NULL unless the statusCode is 202 (NO_CONTENT).
   */
  public final Object body;

  /**
   * The content type to use when serializing the body (null if statusCode is 202)
   */
  public final String contentType;

  KarResponse() {
    this.statusCode = NO_CONTENT;
    this.body = null;
    this.contentType = null;
  }

  KarResponse(int statusCode) {
    this.statusCode = statusCode;
    this.contentType = null;
    this.body = null;
  }

  KarResponse(int statusCode, String contentType, Object body) {
    this.statusCode = statusCode;
    this.contentType = contentType;
    this.body = body;
  }
}
