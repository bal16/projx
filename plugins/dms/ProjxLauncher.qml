import QtQuick
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Services
import qs.Modules.Plugins

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
        if (pluginService) {
            var t = pluginService.loadPluginData(pluginId, "trigger");
            if (t) root.trigger = t;
            var m = pluginService.loadPluginData(pluginId, "maxResults");
            if (m) root.maxResults = parseInt(m);
            var s = pluginService.loadPluginData(pluginId, "showProjects");
            if (s !== null && s !== undefined) root.showProjects = (s === true || s === "true");
            var nw = pluginService.loadPluginData(pluginId, "newWindow");
            if (nw !== null && nw !== undefined) root.newWindow = (nw === true || nw === "true");
        }
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
                root.parseData();
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

            if (action === "scan" || action === "import-zed" || action === "import-all" || action === "add") {
                root.refreshCache();
            }

            if (exitCode !== 0 && ToastService) {
                ToastService.showInfo("pjx: command failed (exit " + exitCode + ")");
            }
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

    function parseData() {
        var lines = root.rawData.split("\n");
        var projects = [];
        for (var i = 0; i < lines.length; i++) {
            var line = lines[i];
            if (line.trim() === "") continue;
            var f = line.split("\x1f");
            if (f.length < 4) continue;
            projects.push({
                id: f[0] || "",
                name: f[1] || "",
                path: f[2] || "",
                editor: f[3] || "",
                tags: f[4] || "",
                group: f[5] || "",
                favorite: f[6] === "1",
                gitBranch: f[7] || "",
                gitDirty: parseInt(f[8]) || 0,
                remote: f[9] || ""
            });
        }
        root.cachedProjects = projects;
    }

    function shortPath(p) {
        if (root.homeDir !== "" && p.indexOf(root.homeDir) === 0) {
            return "~" + p.substring(root.homeDir.length);
        }
        return p;
    }

    function hasTag(p, tag) {
        var parts = (p.tags || "").split(",");
        var needle = (tag || "").toLowerCase();
        for (var i = 0; i < parts.length; i++) {
            if (parts[i].trim().toLowerCase() === needle) return true;
        }
        return false;
    }

    function utilItems(filter) {
        var q = (filter || "").toLowerCase();
        var all = [
            { name: "Scan directories for projects", comment: "run pjx scan on configured roots", action: "util:scan", icon: "material:search", categories: ["projx"] },
            { name: "Import projects from Zed", comment: "one-time migration", action: "util:import-zed", icon: "material:download", categories: ["projx"] },
            { name: "Import from all editors", comment: "zed, vscode, cursor, ...", action: "util:import-all", icon: "material:download", categories: ["projx"] },
            { name: "Set default editor: zed", comment: "pjx config set default_editor zed", action: "util:editor-zed", icon: "material:code", categories: ["projx"] },
            { name: "Set default editor: vscode", comment: "pjx config set default_editor code", action: "util:editor-code", icon: "material:code", categories: ["projx"] },
            { name: "Set default editor: nvim", comment: "pjx config set default_editor nvim", action: "util:editor-nvim", icon: "material:code", categories: ["projx"] },
            { name: "Refresh project list", comment: "clear cache and reload", action: "util:refresh", icon: "material:refresh", categories: ["projx"] },
            { name: "List VS Code profiles", comment: "show known VS Code profiles", action: "util:profiles", icon: "material:person", categories: ["projx"] }
        ];
        var out = [];
        for (var i = 0; i < all.length; i++) {
            var it = all[i];
            if (q === "" || it.name.toLowerCase().indexOf(q) >= 0 || it.comment.toLowerCase().indexOf(q) >= 0) {
                out.push(it);
            }
        }
        return out;
    }

    function getItems(query) {
        var items = [];
        var trimmed = query ? query.trim() : "";
        var lowerQuery = trimmed.toLowerCase();

        var tagFilter = null;
        var textFilter = trimmed;
        if (trimmed.indexOf("#") === 0) {
            tagFilter = trimmed.substring(1);
            textFilter = "";
            lowerQuery = "";
        }

        if (root.showProjects) {
            for (var i = 0; i < root.cachedProjects.length; i++) {
                var p = root.cachedProjects[i];
                if (tagFilter && !hasTag(p, tagFilter)) continue;

                var pname = p.name || "";
                var ppath = p.path || "";
                if (textFilter === "" || pname.toLowerCase().indexOf(lowerQuery) >= 0 ||
                    ppath.toLowerCase().indexOf(lowerQuery) >= 0) {
                    var comment = shortPath(ppath);
                    if (p.gitBranch) {
                        comment += "  [" + p.gitBranch;
                        if (p.gitDirty > 0) comment += " +" + p.gitDirty;
                        comment += "]";
                    }
                    if (p.tags) comment += "  " + p.tags;
                    items.push({
                        name: pname,
                        comment: comment,
                        action: "project:" + ppath,
                        icon: p.favorite ? "material:star" : "material:folder",
                        categories: ["projx"]
                    });
                }
                if (items.length >= root.maxResults) break;
            }
        }

        // Add project from path
        if (trimmed !== "" && (trimmed.indexOf("/") >= 0 || trimmed.indexOf("~") === 0 || trimmed.indexOf(".") === 0)) {
            var expanded = trimmed;
            if (expanded.indexOf("~/") === 0) {
                expanded = root.homeDir + expanded.substring(1);
            }
            var already = false;
            for (var j = 0; j < root.cachedProjects.length; j++) {
                if (root.cachedProjects[j].path === expanded) { already = true; break; }
            }
            if (!already) {
                items.push({
                    name: "Add project: " + (expanded.split("/").pop() || expanded),
                    comment: "create project from " + shortPath(expanded),
                    action: "add:" + expanded,
                    icon: "material:add",
                    categories: ["projx"]
                });
            }
        }

        // Profile create (searchable)
        if (lowerQuery.indexOf("profile") >= 0 || lowerQuery.indexOf("prof") >= 0) {
            var m = trimmed.match(/(?:create|new)\s+(.+)/i);
            var cpname = m ? m[1].trim() : "";
            if (cpname !== "") {
                items.push({ name: "Create VS Code profile: " + cpname, comment: "pjx profiles create " + cpname, action: "vscprofile:create:" + cpname, icon: "material:add", categories: ["projx"] });
            } else {
                items.push({ name: "Create VS Code profile", comment: "type: profile create <name>", action: "vscprofile:createprompt", icon: "material:add", categories: ["projx"] });
            }
        }

        // Utility entries (searchable by title/subtitle)
        var utils = utilItems(textFilter);
        for (var u = 0; u < utils.length; u++) {
            items.push(utils[u]);
        }

        if (root.loadError !== "" && items.length === 0) {
            items.push({ name: "pjx error", comment: root.loadError, action: "", icon: "material:error", categories: ["projx"] });
            return items;
        }

        if (items.length === 0) {
            items.push({ name: "No projects found", comment: "run `pjx scan` to discover projects", action: "", icon: "material:folder_off", categories: ["projx"] });
        }
        return items;
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
            } else if (actionData === "createprompt") {
                if (ToastService) ToastService.showInfo("Type: profile create <name>");
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
