package com.autoscout.burp;

import burp.api.montoya.BurpExtension;
import burp.api.montoya.MontoyaApi;

public class AutoscoutExtension implements BurpExtension {
    @Override
    public void initialize(MontoyaApi api) {
        api.extension().setName("Autoscout Integration");

        // UI Tab
        AutoscoutTab uiTab = new AutoscoutTab(api);
        api.userInterface().registerSuiteTab("Autoscout", uiTab.getUiComponent());

        // Http Handler (Handles Proxy, Repeater, Intruder, etc.)
        AutoscoutHttpHandler httpHandler = new AutoscoutHttpHandler(api, uiTab);
        api.http().registerHttpHandler(httpHandler);

        api.logging().logToOutput("Autoscout Burp Integration loaded successfully.");
    }
}
