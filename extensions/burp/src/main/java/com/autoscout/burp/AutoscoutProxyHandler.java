package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.proxy.ProxyRequestHandler;
import burp.api.montoya.proxy.ProxyRequestReceivedAction;
import burp.api.montoya.proxy.ProxyRequestToBeSentAction;
import burp.api.montoya.proxy.ProxyResponseHandler;
import burp.api.montoya.proxy.ProxyResponseReceivedAction;
import burp.api.montoya.proxy.ProxyResponseToBeSentAction;
import burp.api.montoya.http.message.requests.HttpRequest;
import burp.api.montoya.http.message.responses.HttpResponse;
import burp.api.montoya.core.ByteArray;
import burp.api.montoya.proxy.http.InterceptedRequest;
import burp.api.montoya.proxy.http.InterceptedResponse;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Base64;

public class AutoscoutProxyHandler implements ProxyRequestHandler, ProxyResponseHandler {
    private final MontoyaApi api;
    private final AutoscoutTab ui;
    private final Gson gson;

    public AutoscoutProxyHandler(MontoyaApi api, AutoscoutTab ui) {
        this.api = api;
        this.ui = ui;
        this.gson = new Gson();
    }

    @Override
    public ProxyRequestReceivedAction handleRequestReceived(InterceptedRequest interceptedRequest) {
        if (!ui.isEnabled() || apiEndpointInvalid()) {
            return ProxyRequestReceivedAction.continueWith(interceptedRequest);
        }
        
        HttpRequest modifiedRequest = sendToAutoscout("request", interceptedRequest);
        if (modifiedRequest != null) {
            ui.log("Request modified by Autoscout: " + interceptedRequest.url());
            return ProxyRequestReceivedAction.continueWith(modifiedRequest);
        }
        return ProxyRequestReceivedAction.continueWith(interceptedRequest);
    }

    @Override
    public ProxyRequestToBeSentAction handleRequestToBeSent(InterceptedRequest interceptedRequest) {
        return ProxyRequestToBeSentAction.continueWith(interceptedRequest);
    }

    @Override
    public ProxyResponseReceivedAction handleResponseReceived(InterceptedResponse interceptedResponse) {
        if (!ui.isEnabled() || apiEndpointInvalid()) {
            return ProxyResponseReceivedAction.continueWith(interceptedResponse);
        }
        
        HttpResponse modifiedResponse = sendToAutoscoutResponse("response", interceptedResponse);
        if (modifiedResponse != null) {
            ui.log("Response modified by Autoscout.");
            return ProxyResponseReceivedAction.continueWith(modifiedResponse);
        }
        return ProxyResponseReceivedAction.continueWith(interceptedResponse);
    }

    @Override
    public ProxyResponseToBeSentAction handleResponseToBeSent(InterceptedResponse interceptedResponse) {
        return ProxyResponseToBeSentAction.continueWith(interceptedResponse);
    }

    private boolean apiEndpointInvalid() {
        return ui.getApiEndpoint().trim().isEmpty();
    }

    private HttpRequest sendToAutoscout(String type, InterceptedRequest req) {
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
            // Ignore connection errors to not spam
        }
        return null;
    }

    private HttpResponse sendToAutoscoutResponse(String type, InterceptedResponse resp) {
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
            // Ignore connection errors to not spam
        }
        return null;
    }
}
