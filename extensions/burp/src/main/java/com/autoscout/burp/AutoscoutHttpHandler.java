package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.http.handler.HttpHandler;
import burp.api.montoya.http.handler.HttpRequestToBeSent;
import burp.api.montoya.http.handler.HttpResponseReceived;
import burp.api.montoya.http.handler.RequestToBeSentAction;
import burp.api.montoya.http.handler.ResponseReceivedAction;
import burp.api.montoya.http.message.HttpRequestResponse;
import burp.api.montoya.http.message.requests.HttpRequest;
import burp.api.montoya.http.message.responses.HttpResponse;
import burp.api.montoya.http.message.HttpHeader;
import burp.api.montoya.core.ByteArray;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.ArrayList;
import java.util.List;

public class AutoscoutHttpHandler implements HttpHandler {
    private final MontoyaApi api;
    private final AutoscoutTab ui;
    private final Gson gson;

    public AutoscoutHttpHandler(MontoyaApi api, AutoscoutTab ui) {
        this.api = api;
        this.ui = ui;
        this.gson = new Gson();
    }

    @Override
    public RequestToBeSentAction handleHttpRequestToBeSent(HttpRequestToBeSent httpRequestToBeSent) {
        String toolName = httpRequestToBeSent.toolSource().toolType().name();
        HttpRequest currentRequest = httpRequestToBeSent;

        // Auto-Inject BB Headers for PROXY traffic
        if (toolName.equals("PROXY") && !ui.getBbHeaders().isEmpty()) {
            for (String[] header : ui.getBbHeaders()) {
                currentRequest = currentRequest.withHeader(HttpHeader.httpHeader(header[0], header[1]));
            }
        }

        if (!ui.isEnabled() || !ui.isAutoForwardEnabled() || apiEndpointInvalid()) {
            return RequestToBeSentAction.continueWith(currentRequest);
        }

        // Selective Routing
        if (toolName.equals("PROXY") && !ui.isProxyEnabled()) return RequestToBeSentAction.continueWith(currentRequest);
        if (toolName.equals("REPEATER") && !ui.isRepeaterEnabled()) return RequestToBeSentAction.continueWith(currentRequest);
        if (toolName.equals("INTRUDER") && !ui.isIntruderEnabled()) return RequestToBeSentAction.continueWith(currentRequest);

        HttpRequest modifiedRequest = sendToAutoscout("request", currentRequest, toolName);
        if (modifiedRequest != null) {
            ui.log("[" + toolName + "] Request modified: " + currentRequest.url());
            return RequestToBeSentAction.continueWith(modifiedRequest);
        }
        return RequestToBeSentAction.continueWith(currentRequest);
    }

    @Override
    public ResponseReceivedAction handleHttpResponseReceived(HttpResponseReceived httpResponseReceived) {
        if (!ui.isEnabled() || !ui.isAutoForwardEnabled() || apiEndpointInvalid()) {
            return ResponseReceivedAction.continueWith(httpResponseReceived);
        }

        String toolName = httpResponseReceived.toolSource().toolType().name();
        
        // Selective Routing
        if (toolName.equals("PROXY") && !ui.isProxyEnabled()) return ResponseReceivedAction.continueWith(httpResponseReceived);
        if (toolName.equals("REPEATER") && !ui.isRepeaterEnabled()) return ResponseReceivedAction.continueWith(httpResponseReceived);
        if (toolName.equals("INTRUDER") && !ui.isIntruderEnabled()) return ResponseReceivedAction.continueWith(httpResponseReceived);

        HttpResponse modifiedResponse = sendToAutoscoutResponse("response", httpResponseReceived, toolName);
        if (modifiedResponse != null) {
            ui.log("[" + toolName + "] Response modified.");
            return ResponseReceivedAction.continueWith(modifiedResponse);
        }
        return ResponseReceivedAction.continueWith(httpResponseReceived);
    }

    public void sendManual(HttpRequestResponse message) {
        if (apiEndpointInvalid()) {
            ui.log("[DEBUG] Manual send skipped: API endpoint invalid.");
            return;
        }
        
        String endpoint = ui.getApiEndpoint() + "/manual";
        try {
            ui.log("[DEBUG] Attempting POST to " + endpoint);
            URL url = new URL(endpoint);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setDoOutput(true);

            JsonObject payload = new JsonObject();
            payload.addProperty("url", message.request().url());
            payload.addProperty("method", message.request().method());
            payload.addProperty("tool", "MANUAL");
            payload.addProperty("request_body", Base64.getEncoder().encodeToString(message.request().body().getBytes()));
            
            if (message.hasResponse()) {
                payload.addProperty("status", message.response().statusCode());
                payload.addProperty("response_body", Base64.getEncoder().encodeToString(message.response().body().getBytes()));
            }

            try (OutputStream os = conn.getOutputStream()) {
                byte[] input = gson.toJson(payload).getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            int code = conn.getResponseCode();
            if (code == 200) {
                ui.log("<- Autoscout acknowledged manual analysis.");
            } else {
                ui.log("!! Manual send failed. Server returned: " + code);
            }
        } catch (Exception e) {
            ui.log("!! Manual send failed with exception: " + e.getMessage());
        }
    }

    private boolean apiEndpointInvalid() {
        return ui.getApiEndpoint().trim().isEmpty();
    }

    private HttpRequest sendToAutoscout(String type, HttpRequest req, String tool) {
        try {
            URL url = new URL(ui.getApiEndpoint() + "/request");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setDoOutput(true);
            conn.setConnectTimeout(2000);
            conn.setReadTimeout(2000);

            JsonObject payload = new JsonObject();
            payload.addProperty("url", req.url());
            payload.addProperty("method", req.method());
            payload.addProperty("tool", tool);
            payload.addProperty("body", Base64.getEncoder().encodeToString(req.body().getBytes()));

            try (OutputStream os = conn.getOutputStream()) {
                byte[] input = gson.toJson(payload).getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            if (conn.getResponseCode() == 200) {
                JsonObject responseJson = JsonParser.parseReader(new InputStreamReader(conn.getInputStream())).getAsJsonObject();
                if (responseJson.has("modified") && responseJson.get("modified").getAsBoolean()) {
                    String newBodyBase64 = responseJson.get("body").getAsString();
                    byte[] newBody = Base64.getDecoder().decode(newBodyBase64);
                    return req.withBody(ByteArray.byteArray(newBody));
                }
            }
        } catch (Exception e) {
            // Ignore errors for now
        }
        return null;
    }

    private HttpResponse sendToAutoscoutResponse(String type, HttpResponse resp, String tool) {
        try {
            URL url = new URL(ui.getApiEndpoint() + "/response");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setDoOutput(true);
            conn.setConnectTimeout(2000);
            conn.setReadTimeout(2000);

            JsonObject payload = new JsonObject();
            payload.addProperty("status", resp.statusCode());
            payload.addProperty("tool", tool);
            payload.addProperty("body", Base64.getEncoder().encodeToString(resp.body().getBytes()));

            try (OutputStream os = conn.getOutputStream()) {
                byte[] input = gson.toJson(payload).getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            if (conn.getResponseCode() == 200) {
                JsonObject responseJson = JsonParser.parseReader(new InputStreamReader(conn.getInputStream())).getAsJsonObject();
                if (responseJson.has("modified") && responseJson.get("modified").getAsBoolean()) {
                    String newBodyBase64 = responseJson.get("body").getAsString();
                    byte[] newBody = Base64.getDecoder().decode(newBodyBase64);
                    return resp.withBody(ByteArray.byteArray(newBody));
                }
            }
        } catch (Exception e) {
            // Ignore errors for now
        }
        return null;
    }
}
