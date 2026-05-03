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
    private final JCheckBox enableProxyCheck;
    private final JCheckBox enableRepeaterCheck;
    private final JCheckBox enableIntruderCheck;
    private final JCheckBox autoForwardCheck;

    public AutoscoutTab(MontoyaApi api) {
        mainPanel = new JPanel(new BorderLayout(10, 10));
        mainPanel.setBorder(BorderFactory.createEmptyBorder(10, 10, 10, 10));

        JPanel topPanel = new JPanel(new GridLayout(3, 1, 5, 5));
        
        // Row 1: Global and Endpoint
        JPanel connectionPanel = new JPanel(new FlowLayout(FlowLayout.LEFT, 10, 10));
        enableIntegrationCheck = new JCheckBox("Global Enable");
        enableIntegrationCheck.setSelected(true);
        connectionPanel.add(enableIntegrationCheck);
        connectionPanel.add(new JLabel("Autoscout API:"));
        apiEndpointField = new JTextField("http://127.0.0.1:8081", 20);
        connectionPanel.add(apiEndpointField);
        JButton testBtn = new JButton("Test Connection");
        testBtn.addActionListener(e -> testConnection());
        connectionPanel.add(testBtn);
        topPanel.add(connectionPanel);

        // Row 2: Mode Selection
        JPanel modePanel = new JPanel(new FlowLayout(FlowLayout.LEFT, 10, 0));
        autoForwardCheck = new JCheckBox("Enable Automatic Forwarding", false);
        modePanel.add(autoForwardCheck);
        modePanel.add(new JLabel(" (If unchecked, only 'Send to Autoscout' context menu works)"));
        topPanel.add(modePanel);

        // Row 3: Tool Specific Toggles
        JPanel toolsPanel = new JPanel(new FlowLayout(FlowLayout.LEFT, 10, 0));
        toolsPanel.add(new JLabel("Auto-Forward From:"));
        enableProxyCheck = new JCheckBox("Proxy", true);
        enableRepeaterCheck = new JCheckBox("Repeater", true);
        enableIntruderCheck = new JCheckBox("Intruder", false);
        toolsPanel.add(enableProxyCheck);
        toolsPanel.add(enableRepeaterCheck);
        toolsPanel.add(enableIntruderCheck);
        topPanel.add(toolsPanel);

        mainPanel.add(topPanel, BorderLayout.NORTH);

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
    public boolean isAutoForwardEnabled() { return autoForwardCheck.isSelected(); }
    public boolean isProxyEnabled() { return enableProxyCheck.isSelected(); }
    public boolean isRepeaterEnabled() { return enableRepeaterCheck.isSelected(); }
    public boolean isIntruderEnabled() { return enableIntruderCheck.isSelected(); }
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
