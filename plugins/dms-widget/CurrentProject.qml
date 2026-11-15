import QtQuick
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property var cachedProjects: []
    property string rawData: ""
    property string currentName: "no project"

    property var loadProcess: Process {
        command: ["pjx", "list", "--format=delimited"]
        running: false
        stdout: SplitParser {
            onRead: function(data) { root.rawData += data + "\n"; }
        }
        onExited: function(exitCode, exitStatus) {
            if (exitCode === 0) {
                root.parseData();
                root.updateCurrent();
            }
            root.rawData = "";
        }
    }

    function parseData() {
        var lines = root.rawData.split("\n");
        var projects = [];
        for (var i = 0; i < lines.length; i++) {
            var line = lines[i];
            if (line.trim() === "") continue;
            var f = line.split("\x1f");
            if (f.length < 4) continue;
            projects.push({ name: f[1], path: f[2] });
        }
        root.cachedProjects = projects;
    }

    function updateCurrent() {
        var pwd = Quickshell.env("PWD") || "";
        var best = "";
        var bestLen = 0;
        for (var i = 0; i < root.cachedProjects.length; i++) {
            var p = root.cachedProjects[i];
            if (p.path !== "" && pwd.indexOf(p.path) === 0 && p.path.length > bestLen) {
                best = p.name;
                bestLen = p.path.length;
            }
        }
        root.currentName = best || "no project";
    }

    property var initTimer: Timer {
        interval: 100
        repeat: false
        running: true
        onTriggered: root.loadProcess.running = true
    }

    property var refreshTimer: Timer {
        interval: 5000
        repeat: true
        running: true
        onTriggered: root.loadProcess.running = true
    }

    horizontalBarPill: Component {
        Item {
            implicitWidth: rowLayout.implicitWidth + (Theme.spacing.small * 2)
            implicitHeight: 24

            Row {
                id: rowLayout
                anchors.centerIn: parent
                spacing: 6

                DText {
                    text: root.currentName
                    color: Theme.mOnSurface
                    font.pixelSize: 12
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            MouseArea {
                anchors.fill: parent
                onClicked: {
                    Quickshell.execDetached(["pjx", "tui"]);
                }
            }
        }
    }

    verticalBarPill: Component {
        Item {
            implicitWidth: 24
            implicitHeight: 24

            DText {
                text: "📁"
                anchors.centerIn: parent
                font.pixelSize: 12
            }
        }
    }
}
