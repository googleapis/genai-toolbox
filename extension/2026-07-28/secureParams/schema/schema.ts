// Schema definition for the secureParams extension

export interface ClientCapabilities {
  experimental?: {
    "com.google.cloud/secure-params"?: boolean;
    [key: string]: unknown;
  };
}

export interface Tool {
  name: string;
  description?: string;
  inputSchema: {
    type: "object";
    properties?: Record<string, unknown>;
    required?: string[];
  };
  secureInputSchema?: {
    type: "object";
    properties?: Record<string, unknown>;
    required?: string[];
  };
}

export interface CallToolRequestParams {
  name: string;
  arguments?: Record<string, unknown>;
  secureArguments?: Record<string, string>;
}
