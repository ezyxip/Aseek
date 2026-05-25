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

    BackendProcess {
        id: backend
        Component.onCompleted: start()
    }

    ListModel {
        id: profilesModel
        Component.onCompleted: {
            if (count === 0) {
                append({
                    name: "Profile1",
                    context: "Дефолтный контекст",
                    mcpServers: []
                })
            }
        }
    }
}
