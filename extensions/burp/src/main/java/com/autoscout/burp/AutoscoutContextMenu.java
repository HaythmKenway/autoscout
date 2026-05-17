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
        String context = event.invocationType().name();
        int selectedCount = (event.selectedRequestResponses() != null) ? event.selectedRequestResponses().size() : 0;
        boolean hasEditorMessage = event.messageEditorRequestResponse().isPresent();
        
        ui.log(String.format("[DEBUG] Context Menu in %s. Selected: %d. Has Editor Msg: %b", context, selectedCount, hasEditorMessage));
        
        List<Component> menuItems = new ArrayList<>();

        JMenuItem sendBtn = new JMenuItem("Send to Autoscout");
        sendBtn.addActionListener(e -> sendManualToAutoscout(event));

        menuItems.add(sendBtn);
        return menuItems;
    }

    private void sendManualToAutoscout(ContextMenuEvent event) {
        List<HttpRequestResponse> toSend = new ArrayList<>();
        
        if (event.selectedRequestResponses() != null && !event.selectedRequestResponses().isEmpty()) {
            toSend.addAll(event.selectedRequestResponses());
        } else if (event.messageEditorRequestResponse().isPresent()) {
            toSend.add(event.messageEditorRequestResponse().get().requestResponse());
        }

        if (toSend.isEmpty()) {
            ui.log("[DEBUG] No messages found to send in current context.");
            return;
        }

        ui.log("[DEBUG] Clicked 'Send to Autoscout' for " + toSend.size() + " items.");
        new Thread(() -> {
            for (HttpRequestResponse message : toSend) {
                ui.log("[MANUAL] Sending to Autoscout: " + message.request().url());
                // Always try to include response, but server handles if it's missing
                handler.sendManual(message, true);
            }
        }).start();
    }
}
