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

import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.HttpResponse;

/**
 * Indicates that a request to the attached sidecar has resulted in an
 * unanticipated error of some form.
 */
public class KarSidecarException extends Exception {
  /**
   * The http status code of the response
   */
  public final int statusCode;

  public KarSidecarException(HttpResponse<Buffer> response) {
    super(extractMessage(response));
    this.statusCode = response.statusCode();
  }

  private static String extractMessage(HttpResponse<Buffer> response) {
    String msg = response.statusMessage();
    String body = response.bodyAsString();
    if (body != null && !body.isBlank()) {
      msg = msg + ": " + body;
    }
    return msg;
  }
}
