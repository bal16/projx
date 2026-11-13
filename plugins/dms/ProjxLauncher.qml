import QtQuick
import Quickshell
import Quickshell.Io

QtObject {
    id: root

    property var pluginService: null
    property string trigger: "pjx"

    signal itemsChanged

    property var cachedProjects: []
    property string rawData: ""
    property int maxResults: 50

    property string homeDir: Quickshell.env("HOME") || "/home"

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
        root.itemsChanged();
    }

    // Trigger initial load
    property var initTimer: Timer {
        interval: 100
        repeat: false
        running: true
        onTriggered: root.refreshCache()
    }

    function shortPath(p) {
        if (root.homeDir !== "" && p.indexOf(root.homeDir) === 0) {
            return "~" + p.substring(root.homeDir.length);
        }
        return p;
    }

    function getItems(query) {
        var q = (query || "").trim();
        var ql = q.toLowerCase();
        var results = [];

        // Existing projects
        for (var i = 0; i < root.cachedProjects.length; i++) {
            var p = root.cachedProjects[i];
            var name = p.name || p.path.split("/").pop();
            if (q === "" || name.toLowerCase().indexOf(ql) >= 0 || p.path.toLowerCase().indexOf(ql) >= 0) {
                var subtitle = shortPath(p.path);
                if (p.gitBranch) {
                    subtitle += "  [" + p.gitBranch;
                    if (p.gitDirty > 0) subtitle += " +" + p.gitDirty;
                    subtitle += "]";
                }
                results.push({
                    id: p.path,
                    title: name,
                    subtitle: subtitle,
                    icon: p.favorite ? "star" : "folder"
                });
            }
            if (results.length >= root.maxResults) break;
        }

        // If query looks like a directory path not yet a project, offer to add it.
        if (q !== "" && (q.indexOf("/") >= 0 || q.indexOf("~") === 0 || q.indexOf(".") === 0)) {
            var expanded = q;
            if (expanded.indexOf("~/") === 0) {
                expanded = root.homeDir + expanded.substring(1);
            }
            var already = false;
            for (var j = 0; j < root.cachedProjects.length; j++) {
                if (root.cachedProjects[j].path === expanded) { already = true; break; }
            }
            if (!already) {
                results.unshift({
                    id: "add:" + expanded,
                    title: "Add project: " + expanded.split("/").pop(),
                    subtitle: "create project from " + shortPath(expanded),
                    icon: "plus"
                });
            }
        }

        // Utility entries
        var showUtil = (q === "") || ql.indexOf("scan") >= 0 || ql.indexOf("import") >= 0 ||
            ql.indexOf("config") >= 0 || ql.indexOf("settings") >= 0 || ql.indexOf("editor") >= 0 ||
            ql.indexOf("add") >= 0 || ql.indexOf("help") >= 0;
        if (showUtil) {
            var utils = [
                { id: "util:scan", title: "Scan directories for projects", subtitle: "run pjx scan on configured roots", icon: "folder-search" },
                { id: "util:import-zed", title: "Import projects from Zed", subtitle: "one-time migration", icon: "download" },
                { id: "util:import-all", title: "Import from all editors", subtitle: "zed, vscode, cursor, ...", icon: "download" },
                { id: "util:editor-zed", title: "Set default editor: zed", subtitle: "pjx config set default_editor zed", icon: "code" },
                { id: "util:editor-code", title: "Set default editor: vscode", subtitle: "pjx config set default_editor code", icon: "code" },
                { id: "util:editor-nvim", title: "Set default editor: nvim", subtitle: "pjx config set default_editor nvim", icon: "code" }
            ];
            for (var u = 0; u < utils.length; u++) {
                var ut = utils[u];
                if (q === "" || ut.title.toLowerCase().indexOf(ql) >= 0 || ut.subtitle.toLowerCase().indexOf(ql) >= 0) {
                    results.push(ut);
                }
            }
        }

        if (results.length === 0) {
            results.push({ id: "", title: "No projects found", subtitle: "run `pjx scan` to discover projects", icon: "folder-x" });
        }
        return results;
    }

    function runCmd(args) {
        Quickshell.execDetached(args);
    }

    function openProject(path, newWindow) {
        var args = newWindow ? ["pjx", "open", path, "--new-window"] : ["pjx", "open", path];
        Quickshell.execDetached(args);
    }

    function executeItem(item) {
        if (!item || !item.id) return;
        var id = item.id;
        if (id === "") return;
        if (id.indexOf("add:") === 0) {
            var path = id.substring(4);
            Quickshell.execDetached(["pjx", "add", path]);
            root.refreshCache();
            return;
        }
        if (id.indexOf("util:") === 0) {
            var action = id.substring(6);
            if (action === "scan") {
                Quickshell.execDetached(["pjx", "scan"]);
                root.refreshCache();
            } else if (action === "import-zed") {
                Quickshell.execDetached(["pjx", "import", "zed"]);
                root.refreshCache();
            } else if (action === "import-all") {
                Quickshell.execDetached(["pjx", "import", "--all"]);
                root.refreshCache();
            } else if (action.indexOf("editor-") === 0) {
                var ed = action.substring(7);
                Quickshell.execDetached(["pjx", "config", "set", "default_editor", ed]);
            }
            return;
        }
        // normal project: open
        openProject(id, false);
    }
}
