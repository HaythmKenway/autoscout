package com.autoscout.burp;

import burp.api.montoya.BurpExtension;
import burp.api.montoya.MontoyaApi;

public class AutoscoutExtension implements BurpExtension {
    @Override
    public void initialize(MontoyaApi api) {
        api.extension().setName("Autoscout");

        // UI Tab
        AutoscoutTab uiTab = new AutoscoutTab(api);
        api.userInterface().registerSuiteTab("Autoscout", uiTab.getUiComponent());

        // Http Handler (Handles Proxy, Repeater, Intruder, etc.)
        AutoscoutHttpHandler httpHandler = new AutoscoutHttpHandler(api, uiTab);
        api.http().registerHttpHandler(httpHandler);

        // Context Menu (Right-click "Send to Autoscout")
        AutoscoutContextMenu menuProvider = new AutoscoutContextMenu(api, uiTab, httpHandler);
        api.userInterface().registerContextMenuItemsProvider(menuProvider);

        // API Server for callbacks from Go tool
        AutoscoutCallbackServer callbackServer = new AutoscoutCallbackServer(api, uiTab);
        callbackServer.start(8082); // Dedicated port for tool -> Burp callbacks

        api.logging().logToOutput("Autoscout Burp loaded successfully.");
    }
}
