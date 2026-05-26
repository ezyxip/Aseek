import QtQuick 2.0
import Sailfish.Silica 1.0
import ru.aseek.utils 1.0

ApplicationWindow {
    id: appWindow
    objectName: "applicationWindow"
    initialPage: Qt.resolvedUrl("pages/MainPage.qml")
    cover: Qt.resolvedUrl("cover/DefaultCoverPage.qml")
    allowedOrientations: defaultAllowedOrientations

    property int currentProfileIndex: 0

    onCurrentProfileIndexChanged: {
        if (profilesModel.count > 0 && currentProfileIndex >= 0 && currentProfileIndex < profilesModel.count) {
            if (backend.ready) {
                backend.switchProfile(profilesModel.get(currentProfileIndex).name)
            }
        }
    }

    BackendProcess {
        id: backend
        Component.onCompleted: start()
    }

    Connections {
        target: backend

        onProfilesReceived: {
            profilesModel.clear()
            for (var i = 0; i < profiles.length; i++) {
                var p = profiles[i]
                var servers = []
                if (p.mcpServers) {
                    servers = p.mcpServers
                }
                profilesModel.append({
                    name: p.name || "",
                    context: p.context || "",
                    mcpServers: servers
                })
            }
            if (profilesModel.count === 0) {
                profilesModel.append({
                    name: "Profile1",
                    context: "Дефолтный контекст",
                    mcpServers: []
                })
            }
        }
    }

    ListModel {
        id: profilesModel
    }
}