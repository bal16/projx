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

    // Plugin identity
    property string pluginId: "projxLauncher"

    // Settings (persisted via pluginService)
    property string trigger: "pjx"
    property int maxResults: 50
    property bool showProjects: true

    // Cached data
    property var cachedProjects: []
    property string rawData: ""

    // Home directory
    property string homeDir: Quickshell.env("HOME") || "/home"

    signal itemsChanged

    // Load persisted settings on init
    Component.onCompleted: {
        if (pluginService) {
            var t = pluginService.loadPluginData(pluginId, "trigger");
            if (t) root.trigger = t;
            var m = pluginService.loadPluginData(pluginId, "maxResults");
            if (m) root.maxResults = parseInt(m);
            var s = pluginService.loadPluginData(pluginId, "showProjects");
            if (s !== null && s !== undefined) root.showProjects = (s === true || s === "true");
        }
        root.refreshCache();
    }

    onTriggerChanged: {
        if (pluginService) pluginService.savePluginData(pluginId, "trigger", trigger);
        itemsChanged();
    }

    // Load projects via pjx CLI
    property var loadProcess: Process {
        command: ["pjx", "list", "--format=delimited"]
        running: false
        stdout: SplitParser {
            onRead: function(data) { root.rawData += data; }
        }
        onExited: function(exitCode, exitStatus) {
            if (exitCode === 0) {
                root.parseData();
            }
            root.rawData = "";
            itemsChanged();
        }
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
                id: f[0],
                name: f[1],
                path: f[2],
                editor: f[3],
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

    function getItems(query) {
        var items = [];
        var trimmed = query ? query.trim() : "";
        var lowerQuery = trimmed.toLowerCase();

        // "New empty window" item always on top
        items.push({
            name: "New Empty Window",
            comment: "Open a new empty editor window",
            action: "newwindow",
            icon: "material:add_box",
            categories: ["projx"],
            _preScored: 10000
        });

        if (root.showProjects) {
            for (var i = 0; i < root.cachedProjects.length; i++) {
                var p = root.cachedProjects[i];
                var pname = p.name || "";
                var ppath = p.path || "";

                if (trimmed === "" || pname.toLowerCase().indexOf(lowerQuery) >= 0 ||
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
                        icon: p.favorite ? "material:star" : (p.editor === "zed" ? "material:code" : (p.editor === "code" || p.editor === "codium" || p.editor === "cursor" ? "material:code" : "material:folder")),
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
                    name: "Add project: " + expanded.split("/").pop(),
                    comment: "create project from " + shortPath(expanded),
                    action: "add:" + expanded,
                    icon: "material:add",
                    categories: ["projx"]
                });
            }
        }

        // Utility entries
        var showUtil = trimmed === "" || lowerQuery.indexOf("scan") >= 0 ||
            lowerQuery.indexOf("import") >= 0 || lowerQuery.indexOf("config") >= 0 ||
            lowerQuery.indexOf("settings") >= 0 || lowerQuery.indexOf("editor") >= 0 ||
            lowerQuery.indexOf("add") >= 0 || lowerQuery.indexOf("help") >= 0 ||
            lowerQuery.indexOf("refresh") >= 0;
        if (showUtil) {
            var utils = [
                { name: "Scan directories for projects", comment: "run pjx scan on configured roots", action: "util:scan", icon: "material:search", categories: ["projx"] },
                { name: "Import projects from Zed", comment: "one-time migration", action: "util:import-zed", icon: "material:download", categories: ["projx"] },
                { name: "Import from all editors", comment: "zed, vscode, cursor, ...", action: "util:import-all", icon: "material:download", categories: ["projx"] },
                { name: "Set default editor: zed", comment: "pjx config set default_editor zed", action: "util:editor-zed", icon: "material:code", categories: ["projx"] },
                { name: "Set default editor: vscode", comment: "pjx config set default_editor code", action: "util:editor-code", icon: "material:code", categories: ["projx"] },
                { name: "Set default editor: nvim", comment: "pjx config set default_editor nvim", action: "util:editor-nvim", icon: "material:code", categories: ["projx"] },
                { name: "Refresh project list", comment: "clear cache and reload", action: "util:refresh", icon: "material:refresh", categories: ["projx"] }
            ];
            for (var u = 0; u < utils.length; u++) {
                var ut = utils[u];
                if (trimmed === "" || ut.name.toLowerCase().indexOf(lowerQuery) >= 0 ||
                    ut.comment.toLowerCase().indexOf(lowerQuery) >= 0) {
                    items.push(ut);
                }
            }
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

        if (actionType === "newwindow") {
            Quickshell.execDetached([exe, "-n"]);
            if (ToastService) ToastService.showInfo("Opening new editor window");
        } else if (actionType === "project") {
            Quickshell.execDetached([exe, "open", actionData]);
            if (ToastService) ToastService.showInfo("Opening " + actionData.split("/").pop());
        } else if (actionType === "add") {
            Quickshell.execDetached([exe, "add", actionData]);
            if (ToastService) ToastService.showInfo("Adding project: " + actionData);
            root.refreshCache();
        } else if (actionType === "util") {
            if (actionData === "scan") {
                Quickshell.execDetached([exe, "scan"]);
                if (ToastService) ToastService.showInfo("Scanning directories");
                root.refreshCache();
            } else if (actionData === "import-zed") {
                Quickshell.execDetached([exe, "import", "zed"]);
                if (ToastService) ToastService.showInfo("Importing from Zed");
                root.refreshCache();
            } else if (actionData === "import-all") {
                Quickshell.execDetached([exe, "import", "--all"]);
                if (ToastService) ToastService.showInfo("Importing from all editors");
                root.refreshCache();
            } else if (actionData.indexOf("editor-") === 0) {
                var ed = actionData.substring(7);
                Quickshell.execDetached([exe, "config", "set", "default_editor", ed]);
                if (ToastService) ToastService.showInfo("Default editor set to " + ed);
            } else if (actionData === "refresh") {
                root.refreshCache();
                if (ToastService) ToastService.showInfo("Refreshed");
            }
        }
    }

    // Allow DMS to call refresh
    function refresh() { root.refreshCache(); }
}
