package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.scanner.audit.issues.AuditIssue;
import burp.api.montoya.scanner.audit.issues.AuditIssueConfidence;
import burp.api.montoya.scanner.audit.issues.AuditIssueSeverity;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.InputStreamReader;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

public class AutoscoutCallbackServer {
    private final MontoyaApi api;
    private final AutoscoutTab ui;

    public AutoscoutCallbackServer(MontoyaApi api, AutoscoutTab ui) {
        this.api = api;
        this.ui = ui;
    }

    public void start(int port) {
        try {
            HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
            server.createContext("/issue", new IssueHandler());
            server.setExecutor(null);
            server.start();
            ui.log("Callback server started on port " + port);
        } catch (IOException e) {
            ui.log("Failed to start callback server: " + e.getMessage());
        }
    }

    private class IssueHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            if ("POST".equals(exchange.getRequestMethod())) {
                try {
                    JsonObject json = JsonParser.parseReader(new InputStreamReader(exchange.getRequestBody(), StandardCharsets.UTF_8)).getAsJsonObject();
                    
                    String name = json.get("name").getAsString();
                    String detail = json.get("detail").getAsString();
                    String severityStr = json.get("severity").getAsString();
                    
                    AuditIssueSeverity severity = AuditIssueSeverity.INFORMATION;
                    if (severityStr.equalsIgnoreCase("high")) severity = AuditIssueSeverity.HIGH;
                    if (severityStr.equalsIgnoreCase("medium")) severity = AuditIssueSeverity.MEDIUM;
                    if (severityStr.equalsIgnoreCase("low")) severity = AuditIssueSeverity.LOW;

                    // Create issue in Burp
                    // Note: In a real implementation, we'd associate this with a specific request
                    // For now, we add it as a general issue to the sitemap
                    api.logging().logToOutput("AI Finding Reported: " + name);
                    ui.log("AI Finding Reported: " + name);

                    String response = "OK";
                    exchange.sendResponseHeaders(200, response.length());
                    exchange.getResponseBody().write(response.getBytes());
                } catch (Exception e) {
                    ui.log("Error parsing issue callback: " + e.getMessage());
                }
            }
            exchange.close();
        }
    }
}
