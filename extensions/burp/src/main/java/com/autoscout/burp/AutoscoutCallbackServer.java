package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.scanner.audit.issues.AuditIssueSeverity;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;

public class AutoscoutCallbackServer {
    private final MontoyaApi api;
    private final AutoscoutTab ui;
    private Thread serverThread;

    public AutoscoutCallbackServer(MontoyaApi api, AutoscoutTab ui) {
        this.api = api;
        this.ui = ui;
    }

    public void start(int port) {
        serverThread = new Thread(() -> {
            try (ServerSocket serverSocket = new ServerSocket(port)) {
                ui.log("Callback server started on port " + port);
                while (!Thread.currentThread().isInterrupted()) {
                    try (Socket clientSocket = serverSocket.accept()) {
                        handleClient(clientSocket);
                    } catch (IOException e) {
                        if (!Thread.currentThread().isInterrupted()) {
                            ui.log("Error accepting connection: " + e.getMessage());
                        }
                    }
                }
            } catch (IOException e) {
                ui.log("Failed to start callback server: " + e.getMessage());
            }
        });
        serverThread.setDaemon(true);
        serverThread.start();
    }

    private void handleClient(Socket socket) throws IOException {
        BufferedReader reader = new BufferedReader(new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8));
        String line = reader.readLine();
        if (line == null) return;

        // Basic HTTP Parser
        boolean isPost = line.startsWith("POST /issue");
        int contentLength = 0;

        while (!(line = reader.readLine()).isEmpty()) {
            if (line.toLowerCase().startsWith("content-length:")) {
                contentLength = Integer.parseInt(line.substring(15).trim());
            }
        }

        if (isPost && contentLength > 0) {
            char[] body = new char[contentLength];
            reader.read(body);
            String jsonBody = new String(body);

            try {
                JsonObject json = JsonParser.parseString(jsonBody).getAsJsonObject();
                String name = json.get("name").getAsString();
                String severityStr = json.get("severity").getAsString();

                AuditIssueSeverity severity = AuditIssueSeverity.INFORMATION;
                if (severityStr.equalsIgnoreCase("high")) severity = AuditIssueSeverity.HIGH;
                if (severityStr.equalsIgnoreCase("medium")) severity = AuditIssueSeverity.MEDIUM;
                if (severityStr.equalsIgnoreCase("low")) severity = AuditIssueSeverity.LOW;

                api.logging().logToOutput("AI Finding Reported: " + name);
                ui.log("AI Finding Reported: " + name);

            } catch (Exception e) {
                ui.log("Error parsing issue callback: " + e.getMessage());
            }
        }

        // Send Response
        OutputStream out = socket.getOutputStream();
        String response = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK";
        out.write(response.getBytes(StandardCharsets.UTF_8));
        out.flush();
    }
}
