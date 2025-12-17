import axios, { AxiosError, type AxiosInstance, AxiosRequestConfig, type AxiosResponse } from "axios";
import {
    IRequestResponsePayload,
    type BackendResponse,
    type SuccessResponse,
    type ErrorResponse,
    type OIDCErrorResponse,
} from "./interfaces/IRequestResponsePayload";

interface CacheEntry<D> {
    data: IRequestResponsePayload<D, unknown>;
    timestamp: number;
    ttl?: number;
}

export interface GetOptions {
    useCache?: boolean;
    ttl?: number;
}

/**
 * Determines whether a backend response matches the SuccessResponse shape.
 *
 * @param data - The backend response to check
 * @returns `true` if `data` is a `SuccessResponse` (has `success === true`), `false` otherwise
 */
function isSuccessResponse<T>(data: BackendResponse<T>): data is SuccessResponse<T> {
    return typeof data === "object" && data !== null && "success" in data && data.success === true;
}

/**
 * Determines whether a value matches the ErrorResponse shape.
 *
 * @param data - The value to test
 * @returns `true` if `data` is an object with `success === false` and an `error` field, `false` otherwise.
 */
function isErrorResponse(data: unknown): data is ErrorResponse {
    return typeof data === "object" && data !== null && "success" in data && data.success === false && "error" in data;
}

/**
 * Determines whether a value matches the OIDC error response shape.
 *
 * Recognizes objects that contain an `error` property of type `string` and do not include a `success` property.
 *
 * @param data - The value to test
 * @returns `true` if `data` conforms to `OIDCErrorResponse`, `false` otherwise.
 */
function isOIDCErrorResponse(data: unknown): data is OIDCErrorResponse {
    return (
        typeof data === "object" &&
        data !== null &&
        "error" in data &&
        !("success" in data) &&
        typeof (data as { error: unknown }).error === "string"
    );
}

/**
 * Normalize various backend response shapes into a consistent result carrying either data or error details.
 *
 * @param response - Axios response containing the backend payload to parse
 * @returns An object with `success: true` and `data` (and optional `message`) when the backend indicates success; otherwise `success: false` with `error` and optional `errorDescription` and `errorUri`
 */
function parseBackendResponse<D>(response: AxiosResponse<BackendResponse<D>>): {
    success: boolean;
    data?: D;
    message?: string;
    error?: string;
    errorDescription?: string;
    errorUri?: string;
} {
    const responseData = response.data;

    if (isSuccessResponse(responseData)) {
        return {
            success: true,
            data: responseData.data,
            message: responseData.message,
        };
    }

    if (isErrorResponse(responseData)) {
        return {
            success: false,
            error: responseData.error,
        };
    }

    if (isOIDCErrorResponse(responseData)) {
        return {
            success: false,
            error: responseData.error,
            errorDescription: responseData.error_description,
            errorUri: responseData.error_uri,
        };
    }

    return {
        success: true,
        data: responseData as D,
    };
}

/**
 * Extracts error message from various error formats
 */
export const getApiErrorMessage = (error: AxiosError): string => {
    if (!error.response?.data) {
        return error.message ?? "An unexpected error occurred. Please try again.";
    }

    const data = error.response.data;

    if (isOIDCErrorResponse(data)) {
        return data.error_description ?? data.error;
    }

    if (isErrorResponse(data)) {
        return data.error;
    }

    const legacyError = (data as { readonly error?: { readonly message?: string } })?.error;
    if (legacyError?.message) {
        return legacyError.message;
    }

    return error.message ?? "An unexpected error occurred. Please try again.";
};

export abstract class BaseClient {
    protected abstract get axiosInstance(): AxiosInstance;

    protected static cache = new Map<string, CacheEntry<unknown>>();
    protected static defaultTTL = 5 * 60 * 1000;

    private static generateCacheKey(url: string, config?: AxiosRequestConfig): string {
        const configKey = config
            ? JSON.stringify({
                  params: config.params,
                  headers: config.headers,
              })
            : "";
        return `${url}${configKey}`;
    }

    private static isCacheValid(entry: CacheEntry<unknown>): boolean {
        if (!entry.ttl) return true;
        return Date.now() - entry.timestamp < entry.ttl;
    }

    public static clearCache(url?: string): void {
        if (url) {
            for (const key of BaseClient.cache.keys()) {
                if (key.startsWith(url)) {
                    BaseClient.cache.delete(key);
                }
            }
        } else {
            BaseClient.cache.clear();
        }
    }

    public async get<D = unknown, E = unknown>(
        url: string,
        config?: AxiosRequestConfig,
        options?: GetOptions,
    ): Promise<IRequestResponsePayload<D, E>> {
        const useCache = options?.useCache ?? true;
        const cacheKey = BaseClient.generateCacheKey(url, config);

        if (useCache) {
            const cached = BaseClient.cache.get(cacheKey);
            if (cached && BaseClient.isCacheValid(cached)) {
                return cached.data as IRequestResponsePayload<D, E>;
            }
        }

        try {
            const axiosResponse: AxiosResponse<BackendResponse<D>> = await this.axiosInstance.get(url, config);

            if (axiosResponse.status === 302 || axiosResponse.status === 301) {
                const redirectUrl = axiosResponse.headers.location || axiosResponse.request?.responseURL || url;
                return {
                    success: false,
                    isRedirect: true,
                    redirectUrl,
                    rawResponse: axiosResponse as AxiosResponse<unknown>,
                };
            }

            const parsed = parseBackendResponse(axiosResponse);

            if (parsed.success) {
                const response: IRequestResponsePayload<D, E> = {
                    success: true,
                    data: parsed.data as D,
                    message: parsed.message,
                    rawResponse: axiosResponse,
                };

                if (useCache) {
                    BaseClient.cache.set(cacheKey, {
                        data: response,
                        timestamp: Date.now(),
                        ttl: options?.ttl ?? BaseClient.defaultTTL,
                    });
                }

                return response;
            } else {
                return {
                    success: false,
                    error: parsed.error ?? "An error occurred",
                    errorDescription: parsed.errorDescription,
                    errorUri: parsed.errorUri,
                    rawError: new AxiosError(parsed.error) as AxiosError<E>,
                };
            }
        } catch (error) {
            const axiosError = error as AxiosError<E>;
            const errorData = axiosError.response?.data;

            if (errorData) {
                if (isOIDCErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        errorDescription: errorData.error_description,
                        errorUri: errorData.error_uri,
                        rawError: axiosError,
                    };
                }
                if (isErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        rawError: axiosError,
                    };
                }
            }

            return {
                success: false,
                error: getApiErrorMessage(axiosError),
                rawError: axiosError,
            };
        }
    }

    public async post<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        try {
            const axiosResponse: AxiosResponse<BackendResponse<T>> = await this.axiosInstance.post(url, data, config);

            if (axiosResponse.status === 302 || axiosResponse.status === 301) {
                const redirectUrl = axiosResponse.headers.location || axiosResponse.request?.responseURL || url;
                return {
                    success: false,
                    isRedirect: true,
                    redirectUrl,
                    rawResponse: axiosResponse as AxiosResponse<unknown>,
                };
            }

            const parsed = parseBackendResponse(axiosResponse);

            if (parsed.success) {
                const response: IRequestResponsePayload<T, D> = {
                    success: true,
                    data: parsed.data as T,
                    message: parsed.message,
                    rawResponse: axiosResponse,
                };

                BaseClient.clearCache(url);
                return response;
            } else {
                return {
                    success: false,
                    error: parsed.error ?? "An error occurred",
                    errorDescription: parsed.errorDescription,
                    errorUri: parsed.errorUri,
                    rawError: new AxiosError(parsed.error) as AxiosError<D>,
                };
            }
        } catch (error) {
            const axiosError = error as AxiosError<D>;
            const errorData = axiosError.response?.data;

            if (errorData) {
                if (isOIDCErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        errorDescription: errorData.error_description,
                        errorUri: errorData.error_uri,
                        rawError: axiosError,
                    };
                }
                if (isErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        rawError: axiosError,
                    };
                }
            }

            return {
                success: false,
                error: getApiErrorMessage(axiosError),
                rawError: axiosError,
            };
        }
    }

    public async put<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        try {
            const axiosResponse: AxiosResponse<BackendResponse<T>> = await this.axiosInstance.put(url, data, config);

            if (axiosResponse.status === 302 || axiosResponse.status === 301) {
                const redirectUrl = axiosResponse.headers.location || axiosResponse.request?.responseURL || url;
                return {
                    success: false,
                    isRedirect: true,
                    redirectUrl,
                    rawResponse: axiosResponse as AxiosResponse<unknown>,
                };
            }

            const parsed = parseBackendResponse(axiosResponse);

            if (parsed.success) {
                const response: IRequestResponsePayload<T, D> = {
                    success: true,
                    data: parsed.data as T,
                    message: parsed.message,
                    rawResponse: axiosResponse,
                };

                BaseClient.clearCache(url);
                return response;
            } else {
                return {
                    success: false,
                    error: parsed.error ?? "An error occurred",
                    errorDescription: parsed.errorDescription,
                    errorUri: parsed.errorUri,
                    rawError: new AxiosError(parsed.error) as AxiosError<D>,
                };
            }
        } catch (error) {
            const axiosError = error as AxiosError<D>;
            const errorData = axiosError.response?.data;

            if (errorData) {
                if (isOIDCErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        errorDescription: errorData.error_description,
                        errorUri: errorData.error_uri,
                        rawError: axiosError,
                    };
                }
                if (isErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        rawError: axiosError,
                    };
                }
            }

            return {
                success: false,
                error: getApiErrorMessage(axiosError),
                rawError: axiosError,
            };
        }
    }

    public async patch<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        try {
            const axiosResponse: AxiosResponse<BackendResponse<T>> = await this.axiosInstance.patch(url, data, config);

            if (axiosResponse.status === 302 || axiosResponse.status === 301) {
                const redirectUrl = axiosResponse.headers.location || axiosResponse.request?.responseURL || url;
                return {
                    success: false,
                    isRedirect: true,
                    redirectUrl,
                    rawResponse: axiosResponse as AxiosResponse<unknown>,
                };
            }

            const parsed = parseBackendResponse(axiosResponse);

            if (parsed.success) {
                const response: IRequestResponsePayload<T, D> = {
                    success: true,
                    data: parsed.data as T,
                    message: parsed.message,
                    rawResponse: axiosResponse,
                };

                BaseClient.clearCache(url);
                return response;
            } else {
                return {
                    success: false,
                    error: parsed.error ?? "An error occurred",
                    errorDescription: parsed.errorDescription,
                    errorUri: parsed.errorUri,
                    rawError: new AxiosError(parsed.error) as AxiosError<D>,
                };
            }
        } catch (error) {
            const axiosError = error as AxiosError<D>;
            const errorData = axiosError.response?.data;

            if (errorData) {
                if (isOIDCErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        errorDescription: errorData.error_description,
                        errorUri: errorData.error_uri,
                        rawError: axiosError,
                    };
                }
                if (isErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        rawError: axiosError,
                    };
                }
            }

            return {
                success: false,
                error: getApiErrorMessage(axiosError),
                rawError: axiosError,
            };
        }
    }

    public async delete<T = unknown, D = Error>(
        url: string,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        try {
            const axiosResponse: AxiosResponse<BackendResponse<T>> = await this.axiosInstance.delete(url, config);

            if (axiosResponse.status === 302 || axiosResponse.status === 301) {
                const redirectUrl = axiosResponse.headers.location || axiosResponse.request?.responseURL || url;
                return {
                    success: false,
                    isRedirect: true,
                    redirectUrl,
                    rawResponse: axiosResponse as AxiosResponse<unknown>,
                };
            }

            const parsed = parseBackendResponse(axiosResponse);

            if (parsed.success) {
                const response: IRequestResponsePayload<T, D> = {
                    success: true,
                    data: parsed.data as T,
                    message: parsed.message,
                    rawResponse: axiosResponse,
                };

                BaseClient.clearCache(url);
                return response;
            } else {
                return {
                    success: false,
                    error: parsed.error ?? "An error occurred",
                    errorDescription: parsed.errorDescription,
                    errorUri: parsed.errorUri,
                    rawError: new AxiosError(parsed.error) as AxiosError<D>,
                };
            }
        } catch (error) {
            const axiosError = error as AxiosError<D>;
            const errorData = axiosError.response?.data;

            if (errorData) {
                if (isOIDCErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        errorDescription: errorData.error_description,
                        errorUri: errorData.error_uri,
                        rawError: axiosError,
                    };
                }
                if (isErrorResponse(errorData)) {
                    return {
                        success: false,
                        error: errorData.error,
                        rawError: axiosError,
                    };
                }
            }

            return {
                success: false,
                error: getApiErrorMessage(axiosError),
                rawError: axiosError,
            };
        }
    }
}

export default class GeneralClient extends BaseClient {
    private constructor() {
        super();
    }
    private static axiosInstance: AxiosInstance | null = null;
    private static clientInstance: GeneralClient | null = null;

    protected get axiosInstance(): AxiosInstance {
        return GeneralClient.instance;
    }

    public static get instance(): AxiosInstance {
        if (!GeneralClient.axiosInstance) {
            GeneralClient.axiosInstance = axios.create({
                baseURL: process.env.NEXT_PUBLIC_API_ENDPOINT_URL,
                headers: {
                    "Content-Type": "application/json",
                    Accept: "application/json",
                },
                timeout: 10000,
                withCredentials: true,
            });
        }
        return GeneralClient.axiosInstance;
    }

    public static get client(): GeneralClient {
        if (!GeneralClient.clientInstance) {
            GeneralClient.clientInstance = new GeneralClient();
        }
        return GeneralClient.clientInstance;
    }

    public static async get<D = unknown, E = unknown>(
        url: string,
        config?: AxiosRequestConfig,
        options?: GetOptions,
    ): Promise<IRequestResponsePayload<D, E>> {
        return GeneralClient.client.get(url, config, options);
    }

    public static async post<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        return GeneralClient.client.post(url, data, config);
    }

    public static async put<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        return GeneralClient.client.put(url, data, config);
    }

    public static async patch<T = unknown, D = Error>(
        url: string,
        data?: unknown,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        return GeneralClient.client.patch(url, data, config);
    }

    public static async delete<T = unknown, D = Error>(
        url: string,
        config?: AxiosRequestConfig<unknown>,
    ): Promise<IRequestResponsePayload<T, D>> {
        return GeneralClient.client.delete(url, config);
    }
}