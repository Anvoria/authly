import { AxiosError, type AxiosResponse } from "axios";

/**
 * Standard SuccessResponse format from backend
 */
export interface SuccessResponse<D = unknown> {
    readonly success: true;
    readonly data: D;
    readonly message?: string;
}

/**
 * Standard ErrorResponse format from backend
 */
export interface ErrorResponse {
    readonly success: false;
    readonly error: string;
}

/**
 * OIDC/OAuth2 ErrorResponse format
 */
export interface OIDCErrorResponse {
    readonly error: string;
    readonly error_description?: string;
    readonly error_uri?: string;
}

/**
 * Response type that can be either wrapped or raw JSON
 */
export type BackendResponse<D = unknown> = SuccessResponse<D> | ErrorResponse | OIDCErrorResponse | D; // Raw JSON (for OIDC endpoints)

/**
 * Request response payload that handles all backend response formats
 */
type IRequestResponsePayload<D = unknown, E = unknown> =
    | {
          readonly success: true;
          readonly data: D;
          readonly message?: string;
          readonly rawResponse: AxiosResponse<BackendResponse<D>>;
          readonly isRedirect?: never;
      }
    | {
          readonly success: false;
          readonly error: string;
          readonly errorDescription?: string;
          readonly errorUri?: string;
          readonly rawError: AxiosError<E>;
          readonly isRedirect?: never;
      }
    | {
          readonly success: false;
          readonly isRedirect: true;
          readonly redirectUrl: string;
          readonly rawResponse: AxiosResponse<unknown>;
          readonly error?: never;
          readonly rawError?: never;
      };

export type { IRequestResponsePayload };
