import QtQuick
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginSettings {
    id: root
    pluginId: "projxLauncher"

    StyledText {
        width: parent.width
        text: "projx Project Manager"
        font.pixelSize: Theme.fontSizeLarge
        font.weight: Font.Bold
        color: Theme.surfaceText
    }

    StyledText {
        width: parent.width
        text: "Quick access to projects managed by pjx (zed, vscode, nvim, ...) from the DMS launcher."
        font.pixelSize: Theme.fontSizeSmall
        color: Theme.surfaceVariantText
        wrapMode: Text.WordWrap
    }

    Rectangle {
        width: parent.width
        height: 1
        color: Theme.outline
        opacity: 0.3
    }

    StringSetting {
        settingKey: "trigger"
        label: "Trigger"
        description: "Launcher trigger keyword (default: pjx)"
        placeholder: "pjx"
        defaultValue: "pjx"
    }

    StringSetting {
        settingKey: "maxResults"
        label: "Max Results"
        description: "Maximum number of items shown (5-200)"
        placeholder: "50"
        defaultValue: "50"
    }

    BoolSetting {
        settingKey: "showProjects"
        label: "Show Projects"
        description: "List saved projects in the launcher"
        defaultValue: true
    }

    BoolSetting {
        settingKey: "newWindow"
        label: "Open in New Window"
        description: "Pass --new-window when opening a project"
        defaultValue: false
    }
}
