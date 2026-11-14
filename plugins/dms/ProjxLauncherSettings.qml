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
        defaultValue: "pjx"
    }

    SpinBoxSetting {
        settingKey: "maxResults"
        label: "Max Results"
        description: "Maximum number of items shown"
        defaultValue: 50
        minValue: 5
        maxValue: 200
    }

    BoolSetting {
        settingKey: "showProjects"
        label: "Show Projects"
        description: "List saved projects in the launcher"
        defaultValue: true
    }
}
