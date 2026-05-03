package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.http.handler.HttpHandler;
import burp.api.montoya.http.handler.HttpRequestToBeSent;
import burp.api.montoya.http.handler.HttpResponseReceived;
import burp.api.montoya.http.handler.RequestToBeSentAction;
import burp.api.montoya.http.handler.ResponseReceivedAction;
import burp.api.montoya.http.message.requests.HttpRequest;
import burp.api.montoya.http.message.responses.HttpResponse;
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
        if (!ui.isEnabled() || apiEndpointInvalid()) {
            return RequestToBeSentAction.continueWith(httpRequestToBeSent);
        }

        HttpRequest modifiedRequest = sendToAutoscout("request", httpRequestToBeSent);
        if (modifiedRequest != null) {
            ui.log("Request analyzed/modified: " + httpRequestToBeSent.url());
            return RequestToBeSentAction.continueWith(modifiedRequest);
        }
        return RequestToBeSentAction.continueWith(httpRequestToBeSent);
    }

    @Override
    public ResponseReceivedAction handleHttpResponseReceived(HttpResponseReceived httpResponseReceived) {
        if (!ui.isEnabled() || apiEndpointInvalid()) {
            return ResponseReceivedAction.continueWith(httpResponseReceived);
        }

        HttpResponse modifiedResponse = sendToAutoscoutResponse("response", httpResponseReceived);
        if (modifiedResponse != null) {
            ui.log("Response analyzed/modified.");
            return ResponseReceivedAction.continueWith(modifiedResponse);
        }
        return ResponseReceivedAction.continueWith(httpResponseReceived);
    }

    private boolean apiEndpointInvalid() {
        return ui.getApiEndpoint().trim().isEmpty();
    }

    private HttpRequest sendToAutoscout(String type, HttpRequest req) {
        try {
            ui.log("-> Intercepted " + type + ": " + req.url());
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
                    ui.log("<- Received modified body from Autoscout");
                    return req.withBody(ByteArray.byteArray(newBody));
                }
            } else {
                ui.log("!! Autoscout returned error: " + conn.getResponseCode());
            }
        } catch (Exception e) {
            ui.log("Error sending request to Autoscout: " + e.getMessage());
        }
        return null;
    }

    private HttpResponse sendToAutoscoutResponse(String type, HttpResponse resp) {
        try {
            ui.log("-> Intercepted " + type);
            URL url = new URL(ui.getApiEndpoint() + "/response");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setDoOutput(true);
            conn.setConnectTimeout(2000);
            conn.setReadTimeout(2000);

            JsonObject payload = new JsonObject();
            payload.addProperty("status", resp.statusCode());
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
                    ui.log("<- Received modified response body from Autoscout");
                    return resp.withBody(ByteArray.byteArray(newBody));
                }
            } else {
                ui.log("!! Autoscout returned error: " + conn.getResponseCode());
            }
        } catch (Exception e) {
            ui.log("Error sending response to Autoscout: " + e.getMessage());
        }
        return null;
    }
}
