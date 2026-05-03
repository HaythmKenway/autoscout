package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import javax.swing.*;
import java.awt.*;
import java.net.HttpURLConnection;
import java.net.URL;

public class AutoscoutTab {
    private final JPanel mainPanel;
    private final JTextArea logArea;
    private final JTextField apiEndpointField;
    private final JCheckBox enableIntegrationCheck;

    public AutoscoutTab(MontoyaApi api) {
        mainPanel = new JPanel(new BorderLayout(10, 10));
        mainPanel.setBorder(BorderFactory.createEmptyBorder(10, 10, 10, 10));

        JPanel controlPanel = new JPanel(new FlowLayout(FlowLayout.LEFT, 10, 10));
        enableIntegrationCheck = new JCheckBox("Enable Autoscout Routing");
        controlPanel.add(enableIntegrationCheck);
        controlPanel.add(new JLabel("Autoscout API:"));
        apiEndpointField = new JTextField("http://127.0.0.1:8081", 30);
        controlPanel.add(apiEndpointField);

        JButton testBtn = new JButton("Test Connection");
        testBtn.addActionListener(e -> testConnection());
        controlPanel.add(testBtn);

        mainPanel.add(controlPanel, BorderLayout.NORTH);

        logArea = new JTextArea();
        logArea.setEditable(false);
        JScrollPane scrollPane = new JScrollPane(logArea);
        JPanel logPanel = new JPanel(new BorderLayout());
        logPanel.setBorder(BorderFactory.createTitledBorder("Integration Logs"));
        logPanel.add(scrollPane, BorderLayout.CENTER);
        mainPanel.add(logPanel, BorderLayout.CENTER);
    }

    public Component getUiComponent() { return mainPanel; }
    public boolean isEnabled() { return enableIntegrationCheck.isSelected(); }
    public String getApiEndpoint() { return apiEndpointField.getText(); }
    public void log(String message) {
        SwingUtilities.invokeLater(() -> {
            logArea.append(message + "\n");
        });
    }

    private void testConnection() {
        new Thread(() -> {
            try {
                log("Testing connection to " + getApiEndpoint() + "...");
                URL url = new URL(getApiEndpoint());
                HttpURLConnection conn = (HttpURLConnection) url.openConnection();
                conn.setConnectTimeout(2000);
                conn.setRequestMethod("GET");
                int code = conn.getResponseCode();
                if (code == 200) {
                    log("SUCCESS: Autoscout API is reachable!");
                } else {
                    log("FAILED: Server returned code " + code);
                }
            } catch (Exception ex) {
                log("FAILED: " + ex.getMessage());
            }
        }).start();
    }
}
