import QtQuick
import Quickshell
import Quickshell.Io

QtObject {
    id: root

    property var pluginService: null
    property string trigger: "pm"

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

    function handleQuery(query) {
        var q = (query || "").trim().toLowerCase();
        var results = [];
        for (var i = 0; i < root.cachedProjects.length; i++) {
            var p = root.cachedProjects[i];
            var name = p.name || p.path.split("/").pop();
            if (q === "" || name.toLowerCase().indexOf(q) >= 0 || p.path.toLowerCase().indexOf(q) >= 0) {
                var subtitle = p.path;
                if (root.homeDir !== "" && p.path.indexOf(root.homeDir) === 0) {
                    subtitle = "~" + p.path.substring(root.homeDir.length);
                }
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
        return results;
    }

    function openProject(path, newWindow) {
        var args = newWindow ? ["pjx", "open", path, "--new-window"] : ["pjx", "open", path];
        Quickshell.execDetached(args);
    }
}
