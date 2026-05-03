package com.autoscout.burp;

import burp.api.montoya.MontoyaApi;
import burp.api.montoya.ui.contextmenu.ContextMenuEvent;
import burp.api.montoya.ui.contextmenu.ContextMenuItemsProvider;
import burp.api.montoya.http.message.HttpRequestResponse;

import javax.swing.*;
import java.awt.*;
import java.util.ArrayList;
import java.util.List;

public class AutoscoutContextMenu implements ContextMenuItemsProvider {
    private final MontoyaApi api;
    private final AutoscoutTab ui;
    private final AutoscoutHttpHandler handler;

    public AutoscoutContextMenu(MontoyaApi api, AutoscoutTab ui, AutoscoutHttpHandler handler) {
        this.api = api;
        this.ui = ui;
        this.handler = handler;
    }

    @Override
    public List<Component> provideMenuItems(ContextMenuEvent event) {
        List<Component> menuItems = new ArrayList<>();

        JMenuItem sendToAutoscout = new JMenuItem("Send to Autoscout");
        sendToAutoscout.addActionListener(e -> {
            List<HttpRequestResponse> selectedMessages = event.selectedRequestResponses();
            if (selectedMessages == null || selectedMessages.isEmpty()) {
                return;
            }

            new Thread(() -> {
                for (HttpRequestResponse message : selectedMessages) {
                    ui.log("[MANUAL] Sending selected request to Autoscout: " + message.request().url());
                    handler.sendManual(message);
                }
            }).start();
        });

        menuItems.add(sendToAutoscout);
        return menuItems;
    }
}
