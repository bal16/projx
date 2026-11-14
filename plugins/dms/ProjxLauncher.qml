import QtQuick
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Services
import qs.Modules.Plugins

import "ProjxLogic.js" as Logic

QtObject {
    id: root

    property var pluginService: null
    property string pluginId: "projxLauncher"

    // Settings
    property string trigger: "pjx"
    property int maxResults: 50
    property bool showProjects: true
    property bool newWindow: false

    // Cached data
    property var cachedProjects: []
    property string rawData: ""
    property string loadError: ""

    property string homeDir: Quickshell.env("HOME") || "/home"

    // Action process (for commands where we need output / post-run refresh)
    property string actionOut: ""
    property string pendingAction: ""

    signal itemsChanged

    Component.onCompleted: {
        if (!pluginService)
            return root.refreshCache();
        var t = pluginService.loadPluginData(pluginId, "trigger");
        if (t) root.trigger = t;
        var m = pluginService.loadPluginData(pluginId, "maxResults");
        if (m) root.maxResults = parseInt(m);
        var s = pluginService.loadPluginData(pluginId, "showProjects");
        if (s !== null && s !== undefined) root.showProjects = (s === true || s === "true");
        var nw = pluginService.loadPluginData(pluginId, "newWindow");
        if (nw !== null && nw !== undefined) root.newWindow = (nw === true || nw === "true");
        root.refreshCache();
    }

    onTriggerChanged: {
        if (pluginService) pluginService.savePluginData(pluginId, "trigger", trigger);
        itemsChanged();
    }

    onNewWindowChanged: {
        if (pluginService) pluginService.savePluginData(pluginId, "newWindow", root.newWindow);
    }

    property var loadProcess: Process {
        command: ["pjx", "list", "--format=delimited"]
        running: false
        stdout: SplitParser {
            onRead: function(data) { root.rawData += data; }
        }
        onExited: function(exitCode, exitStatus) {
            if (exitCode === 0) {
                root.loadError = "";
                root.cachedProjects = Logic.parseProjects(root.rawData);
            } else {
                root.loadError = "pjx list failed (exit " + exitCode + ")";
                root.cachedProjects = [];
            }
            root.rawData = "";
            itemsChanged();
        }
    }

    property var actionProcess: Process {
        running: false
        stdout: SplitParser {
            onRead: function(data) { root.actionOut += data; }
        }
        onExited: function(exitCode, exitStatus) {
            var out = (root.actionOut || "").trim();
            var action = root.pendingAction;
            root.actionOut = "";
            root.pendingAction = "";

            if (action === "profiles") {
                if (out === "") out = "no profiles found";
                if (ToastService) ToastService.showInfo(out.split("\n").join(", ").substring(0, 200));
                return;
            }
            if (action === "scan" || action === "import-zed" || action === "import-all" || action === "import-project-manager" || action === "add")
                root.refreshCache();
            if (exitCode !== 0 && ToastService)
                ToastService.showInfo("pjx: command failed (exit " + exitCode + ")");
        }
    }

    function runAction(args, pending) {
        root.pendingAction = pending || "";
        root.actionOut = "";
        actionProcess.command = args;
        actionProcess.running = true;
    }

    function refreshCache() {
        root.rawData = "";
        root.loadProcess.running = true;
    }

    function getItems(query) {
        return Logic.buildItems(root.cachedProjects, query, root.showProjects, root.maxResults, root.homeDir, root.loadError);
    }

    function executeItem(item) {
        if (!item || !item.action) return;
        var action = item.action;
        var colonIndex = action.indexOf(":");
        var actionType = colonIndex > 0 ? action.substring(0, colonIndex) : action;
        var actionData = colonIndex > 0 ? action.substring(colonIndex + 1) : "";
        var exe = "pjx";

        if (actionType === "project") {
            var args = [exe, "open", actionData];
            if (root.newWindow) args.push("--new-window");
            Quickshell.execDetached(args);
            if (ToastService) ToastService.showInfo("Opening " + (actionData.split("/").pop() || actionData));
            return;
        }
        if (actionType === "add") {
            runAction([exe, "add", actionData], "add");
            if (ToastService) ToastService.showInfo("Adding project: " + actionData);
            return;
        }
        if (actionType === "vscprofile") {
            if (actionData.indexOf("create:") === 0) {
                var pname = actionData.substring(7);
                Quickshell.execDetached([exe, "profiles", "create", pname]);
                if (ToastService) ToastService.showInfo("Creating profile: " + pname);
            } else if (actionData === "createprompt" && ToastService) {
                ToastService.showInfo("Type: profile create <name>");
            }
            return;
        }
        if (actionType === "util") {
            if (actionData === "scan") {
                runAction([exe, "scan"], "scan");
                if (ToastService) ToastService.showInfo("Scanning directories");
            } else if (actionData === "import-zed") {
                runAction([exe, "import", "zed"], "import-zed");
                if (ToastService) ToastService.showInfo("Importing from Zed");
            } else if (actionData === "import-all") {
                runAction([exe, "import", "--all"], "import-all");
                if (ToastService) ToastService.showInfo("Importing from all editors");
            } else if (actionData === "import-project-manager") {
                runAction([exe, "import", "project-manager"], "import-project-manager");
                if (ToastService) ToastService.showInfo("Importing from Project Manager");
            } else if (actionData.indexOf("editor-") === 0) {
                var ed = actionData.substring(7);
                Quickshell.execDetached([exe, "config", "set", "default_editor", ed]);
                if (ToastService) ToastService.showInfo("Default editor set to " + ed);
            } else if (actionData === "refresh") {
                root.refreshCache();
                if (ToastService) ToastService.showInfo("Refreshed");
            } else if (actionData === "profiles") {
                runAction([exe, "profiles"], "profiles");
            }
            return;
        }
    }

    function refresh() { root.refreshCache(); }
}
