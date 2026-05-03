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

        // Proxy Handler
        AutoscoutProxyHandler proxyHandler = new AutoscoutProxyHandler(api, uiTab);
        api.proxy().registerRequestHandler(proxyHandler);
        api.proxy().registerResponseHandler(proxyHandler);

        api.logging().logToOutput("Autoscout Burp Integration loaded successfully.");
    }
}
