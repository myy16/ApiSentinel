const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";

interface FetchOptions extends RequestInit {
  token?: string | null;
  organizationId?: string | null;
}

export class ApiError extends Error {
  code: string;
  statusCode: number;
  requestId?: string;

  constructor(message: string, code: string, statusCode: number, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.statusCode = statusCode;
    this.requestId = requestId;
  }
}

export async function apiFetch<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
  const { token, organizationId, headers = {}, ...rest } = options;

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...(headers as Record<string, string>),
  };

  if (token) {
    requestHeaders["Authorization"] = `Bearer ${token}`;
  }

  if (organizationId) {
    requestHeaders["x-organization-id"] = organizationId;
  }

  const url = `${API_BASE_URL}${endpoint.startsWith("/") ? endpoint : `/${endpoint}`}`;

  const response = await fetch(url, {
    credentials: "include",
    headers: requestHeaders,
    ...rest,
  });

  const data = await response.json().catch(() => null);

  if (!response.ok) {
    const errorData = data?.error;

    // If token expired, clear local session and redirect to login
    if (response.status === 401 && typeof window !== "undefined") {
      const currentPath = window.location.pathname;
      if (currentPath !== "/login" && currentPath !== "/register") {
        localStorage.removeItem("apisentinel_access_token");
        localStorage.removeItem("apisentinel_user");
        localStorage.removeItem("apisentinel_org");
        window.location.href = "/login";
      }
    }

    throw new ApiError(
      errorData?.message || `Request failed with status ${response.status}`,
      errorData?.code || "REQUEST_FAILED",
      response.status,
      errorData?.requestId
    );
  }

  return data as T;
}
